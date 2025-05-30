package greenplum

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/wal-g/wal-g/internal/databases/postgres"
	"github.com/wal-g/wal-g/internal/multistorage"

	"golang.org/x/sync/errgroup"

	"github.com/wal-g/tracelog"

	"github.com/wal-g/wal-g/internal"
	conf "github.com/wal-g/wal-g/internal/config"
	"github.com/wal-g/wal-g/pkg/storages/storage"
	"github.com/wal-g/wal-g/utility"
)

type DeleteArgs struct {
	Confirmed bool
	FindFull  bool
	Force     bool
}

type DeleteHandler struct {
	internal.DeleteHandler
	permanentBackups []string
	args             DeleteArgs
}

func NewDeleteHandler(folder storage.Folder, args DeleteArgs) (*DeleteHandler, error) {
	backupSentinelObjects, err := internal.GetBackupSentinelObjects(folder)
	if err != nil {
		return nil, err
	}

	backupObjects, err := makeBackupObjects(folder, backupSentinelObjects)
	if err != nil {
		return nil, err
	}

	// todo better lessfunc
	gpLessFunc := func(obj1, obj2 storage.Object) bool {
		return obj1.GetLastModified().Before(obj2.GetLastModified())
	}

	permanentBackups := internal.GetPermanentBackupsFromStorage(folder.GetSubFolder(utility.BaseBackupPath),
		NewGenericMetaFetcher())
	permanentBackupNames := make([]string, 0, len(permanentBackups))
	for name := range permanentBackups {
		permanentBackupNames = append(permanentBackupNames, name)
	}
	isPermanentFunc := func(obj storage.Object) bool {
		return internal.IsPermanent(obj.GetName(), permanentBackups, BackupNameLength)
	}

	return &DeleteHandler{
		DeleteHandler: *internal.NewDeleteHandler(
			folder,
			backupObjects,
			gpLessFunc,
			internal.IsPermanentFunc(isPermanentFunc),
		),
		permanentBackups: permanentBackupNames,
		args:             args,
	}, nil
}

func (h *DeleteHandler) HandleDeleteBefore(args []string) {
	modifier, beforeStr := internal.ExtractDeleteModifierFromArgs(args)

	target, err := h.FindTargetBefore(beforeStr, modifier)
	tracelog.ErrorLogger.FatalOnError(err)
	if target == nil {
		tracelog.InfoLogger.Printf("No backup found for deletion")
		os.Exit(0)
	}

	err = h.DeleteBeforeTarget(target)
	tracelog.ErrorLogger.FatalOnError(err)
}

func (h *DeleteHandler) HandleDeleteRetain(args []string) {
	modifier, retentionStr := internal.ExtractDeleteModifierFromArgs(args)
	retentionCount, err := strconv.Atoi(retentionStr)
	tracelog.ErrorLogger.FatalOnError(err)

	target, err := h.FindTargetRetain(retentionCount, modifier)
	tracelog.ErrorLogger.FatalOnError(err)
	if target == nil {
		tracelog.InfoLogger.Printf("No backup found for deletion")
		os.Exit(0)
	}

	err = h.DeleteBeforeTarget(target)
	tracelog.ErrorLogger.FatalOnError(err)
}

func (h *DeleteHandler) HandleDeleteRetainAfter(args []string) {
	modifier, retentionSir, afterStr := internal.ExtractDeleteRetainAfterModifierFromArgs(args)
	retentionCount, err := strconv.Atoi(retentionSir)
	tracelog.ErrorLogger.FatalOnError(err)

	target, err := h.FindTargetRetainAfter(retentionCount, afterStr, modifier)
	tracelog.ErrorLogger.FatalOnError(err)

	if target == nil {
		tracelog.InfoLogger.Printf("No backup found for deletion")
		os.Exit(0)
	}

	err = h.DeleteBeforeTarget(target)
	tracelog.ErrorLogger.FatalOnError(err)
}

func (h *DeleteHandler) HandleDeleteEverything(args []string) {
	h.DeleteHandler.HandleDeleteEverything(args, h.permanentBackups, h.args.Confirmed)
}

func (h *DeleteHandler) DeleteBeforeTarget(target internal.BackupObject) error {
	tracelog.InfoLogger.Println("Deleting the segments backups...")
	err := h.dispatchDeleteCmd(target, SegDeleteBefore)
	if err != nil {
		return fmt.Errorf("failed to delete the segments backups: %w", err)
	}
	tracelog.InfoLogger.Printf("Finished deleting the segments backups")

	objFilter := func(object storage.Object) bool { return true }
	folderFilter := func(name string) bool { return strings.HasPrefix(name, utility.BaseBackupPath) }
	return h.DeleteBeforeTargetWhere(target, h.args.Confirmed, objFilter, folderFilter)
}

func (h *DeleteHandler) HandleDeleteTarget(targetSelector internal.BackupSelector) {
	target, err := h.FindTargetBySelector(targetSelector)
	tracelog.ErrorLogger.FatalOnError(err)

	if target == nil {
		// since we want to delete the target backup, we should fail if
		// we didn't find the requested backup for deletion
		tracelog.ErrorLogger.Fatal("Requested backup was not found")
	}

	tracelog.InfoLogger.Println("Deleting the segments backups...")
	err = h.dispatchDeleteCmd(target, SegDeleteTarget)
	if err != nil {
		tracelog.ErrorLogger.Fatalf("Failed to delete the segments backups: %v", err)
	}
	tracelog.InfoLogger.Printf("Finished deleting the segments backups")

	folderFilter := func(name string) bool { return true }
	err = h.DeleteTarget(target, h.args.Confirmed, h.args.FindFull, folderFilter)
	tracelog.ErrorLogger.FatalOnError(err)
}

func (h *DeleteHandler) dispatchDeleteCmd(target internal.BackupObject, delType SegDeleteType) error {
	backup, err := NewBackupInStorage(h.Folder, target.GetBackupName(), multistorage.GetStorage(target))
	if err != nil {
		return err
	}
	sentinel, err := backup.GetSentinel()
	if err != nil {
		return fmt.Errorf("failed to load backup %s sentinel: %v", backup.Name, err)
	}

	deleteConcurrency, err := conf.GetMaxConcurrency(conf.GPDeleteConcurrency)
	if err != nil {
		tracelog.WarningLogger.Printf("config error: %v", err)
	}

	errorGroup, _ := errgroup.WithContext(context.Background())
	errorGroup.SetLimit(deleteConcurrency)

	// clean the segments
	for i := range sentinel.Segments {
		meta := sentinel.Segments[i]
		tracelog.InfoLogger.Printf("Processing segment %d (backupId=%s)", meta.ContentID, meta.BackupID)

		errorGroup.Go(func() error {
			segHandler, err := NewSegDeleteHandler(h.Folder, meta.ContentID, h.args, delType)
			if err != nil {
				return err
			}
			segBackup, err := backup.GetSegmentBackup(meta.BackupID, meta.ContentID)
			if err != nil {
				if h.args.Force {
					tracelog.ErrorLogger.Printf("Processing segment %d (backupId=%s): %v", meta.ContentID, meta.BackupID, err)
					return nil // skip non-critical errors in garbage deletion
				}
				return err
			}
			deleteErr := segHandler.Delete(segBackup)
			if deleteErr != nil {
				return deleteErr
			}
			return nil
		})
	}

	return errorGroup.Wait()
}

// HandleDeleteGarbage delete outdated WAL archives and leftover backup files
func (h *DeleteHandler) HandleDeleteGarbage(args []string) error {
	predicate := postgres.ExtractDeleteGarbagePredicate(args)
	backupSelector := internal.NewOldestNonPermanentSelector(NewGenericMetaFetcher())
	oldestBackup, err := backupSelector.Select(h.Folder)
	if err != nil {
		if _, ok := err.(internal.NoBackupsFoundError); ok {
			tracelog.InfoLogger.Println("Couldn't find any non-permanent backups in storage. Not doing anything.")
			return nil
		}
		return err
	}

	target, err := h.FindTargetByName(oldestBackup.Name)
	if err != nil {
		return err
	}

	tracelog.InfoLogger.Println("Processing the segments...")
	err = h.dispatchDeleteCmd(target, SegDeleteBefore)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	tracelog.InfoLogger.Printf("Finished processing the segments backups")

	folderFilter := func(name string) bool { return strings.HasPrefix(name, utility.BaseBackupPath) }
	return h.DeleteBeforeTargetWhere(target, h.args.Confirmed, predicate, folderFilter)
}
