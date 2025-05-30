package postgres

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/wal-g/wal-g/internal"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wal-g/tracelog"
	conf "github.com/wal-g/wal-g/internal/config"
	pg_errors "github.com/wal-g/wal-g/internal/databases/postgres/errors"
	"github.com/wal-g/wal-g/internal/fsutil"
	"github.com/wal-g/wal-g/utility"
)

// TODO : unit tests
// HandleWALPrefetch is invoked by wal-fetch command to speed up database restoration
func HandleWALPrefetch(folderReader internal.StorageFolderReader, walFileName string, location string) error {
	var fileName = walFileName
	location = path.Dir(location)
	waitGroup := &sync.WaitGroup{}
	concurrency, err := conf.GetMaxDownloadConcurrency()
	if err != nil {
		return fmt.Errorf("get max concurrency: %v", err)
	}

	for i := 0; i < concurrency; i++ {
		fileName, err = GetNextWalFilename(fileName)
		if err != nil {
			return fmt.Errorf("get next filename: %v", err)
		}
		waitGroup.Add(1)
		go prefetchFile(location, folderReader.SubFolder(utility.WalPath), fileName, waitGroup)

		prefaultStartLsn, shouldPrefault, timelineID, err := shouldPrefault(fileName)
		if err != nil {
			tracelog.ErrorLogger.Println("ShouldPrefault failed: ", err, " file: ", fileName)
		}
		if shouldPrefault {
			waitGroup.Add(1)
			go prefaultData(prefaultStartLsn, timelineID, waitGroup, folderReader)
		}

		time.Sleep(10 * time.Millisecond) // ramp up in order
	}

	go CleanupPrefetchDirectories(walFileName, location, fsutil.FileSystemCleaner{})

	waitGroup.Wait()
	return nil
}

// TODO : unit tests
func prefaultData(prefaultStartLsn LSN, timelineID uint32, waitGroup *sync.WaitGroup, folderReader internal.StorageFolderReader) {
	defer func() {
		if r := recover(); r != nil {
			tracelog.ErrorLogger.Println("Prefault unsuccessful ", prefaultStartLsn)
		}
		waitGroup.Done()
	}()

	useWalDelta, deltaDataFolder, err := configureWalDeltaUsage()
	if err != nil || !useWalDelta {
		tracelog.DebugLogger.Printf("configure WAL Delta usage: %v", err)
		return
	}

	archiveDirectory := deltaDataFolder.(*fsutil.DiskDataFolder).Path
	archiveDirectory = filepath.Dir(archiveDirectory)
	archiveDirectory = filepath.Dir(archiveDirectory)
	bundle := NewBundle(archiveDirectory, nil, "", &prefaultStartLsn, nil,
		false, viper.GetInt64(conf.TarSizeThresholdSetting))
	bundle.Timeline = timelineID
	startLsn := prefaultStartLsn + LSN(WalSegmentSize*WalFileInDelta)
	err = bundle.DownloadDeltaMap(folderReader.SubFolder(utility.WalPath), startLsn)
	if err != nil {
		tracelog.ErrorLogger.Printf("Error during loading delta map: '%+v'.", err)
		return
	}
	// Start a new tar bundle, walk the archiveDirectory and upload everything there.
	err = bundle.StartQueue(internal.NewNopTarBallMaker())
	if err != nil {
		tracelog.ErrorLogger.Printf("Error during starting tar queue: '%+v'.", err)
		return
	}
	tracelog.InfoLogger.Println("Walking for prefault...")
	err = filepath.Walk(archiveDirectory, bundle.prefaultWalkedFSObject)
	tracelog.ErrorLogger.FatalOnError(err)
	err = bundle.FinishQueue()
	tracelog.ErrorLogger.FatalOnError(err)
}

// TODO : unit tests
func (bundle *Bundle) prefaultWalkedFSObject(path string, info os.FileInfo, err error) error {
	if err != nil {
		if os.IsNotExist(err) {
			tracelog.WarningLogger.Println(path, " deleted during filepath walk")
			return nil
		}
		return err
	}

	if info.Name() != PgControl {
		err = bundle.prefaultHandleTar(path, info)
		if err != nil {
			if err == filepath.SkipDir {
				return err
			}
			return errors.Wrap(err, "HandleWalkedFSObject: handle tar failed")
		}
	}
	return nil
}

// TODO : unit tests
func (bundle *Bundle) prefaultHandleTar(path string, info os.FileInfo) error {
	fileName := info.Name()
	_, excluded := ExcludedFilenames[fileName]
	isDir := info.IsDir()

	if excluded && !isDir {
		return nil
	}

	fileInfoHeader, err := tar.FileInfoHeader(info, fileName)
	if err != nil {
		return errors.Wrap(err, "addToBundle: could not grab header info")
	}

	fileInfoHeader.Name = bundle.GetFileRelPath(path)

	if !excluded && info.Mode().IsRegular() {
		tarBall := bundle.TarBallQueue.Deque()
		tarBall.SetUp(nil)
		go func() {
			err := bundle.prefaultFile(path, info, fileInfoHeader)
			if err != nil {
				panic(err)
			}
			err = bundle.TarBallQueue.CheckSizeAndEnqueueBack(tarBall)
			if err != nil {
				panic(err)
			}
		}()
	} else {
		if excluded && isDir {
			return filepath.SkipDir
		}
	}

	return nil
}

// TODO : unit tests
func (bundle *Bundle) prefaultFile(path string, info os.FileInfo, fileInfoHeader *tar.Header) error {
	incrementBaseLsn := bundle.getIncrementBaseLsn()
	isIncremented := isPagedFile(info, path)
	var fileReader io.ReadCloser
	if isIncremented {
		bitmap, err := bundle.getDeltaBitmapFor(path)
		if _, ok := err.(NoBitmapFoundError); !ok { // this file has changed after the start of backup, so just skip it
			if err != nil {
				return errors.Wrapf(err, "packFileIntoTar: failed to find corresponding bitmap '%s'\n", path)
			}
			tracelog.InfoLogger.Println("Prefaulting ", path)
			fileReader, fileInfoHeader.Size, err = ReadIncrementalFile(path, info.Size(), *incrementBaseLsn, bitmap)
			if _, ok := err.(pg_errors.InvalidBlockError); ok {
				return nil
			} else if err != nil {
				return errors.Wrapf(err, "packFileIntoTar: failed reading incremental file '%s'\n", path)
			}

			_, err := io.Copy(io.Discard, fileReader)

			if err != nil {
				return errors.Wrap(err, "packFileIntoTar: operation failed")
			}
			fileReader.Close()
		}
	}

	return nil
}

// TODO : unit tests
func prefetchFile(location string, reader internal.StorageFolderReader, walFileName string, waitGroup *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			tracelog.ErrorLogger.Println("WAL-prefetch unsuccessful ", walFileName, r)
		}
		waitGroup.Done()
	}()

	_, runningLocation, oldPath, newPath := getPrefetchLocations(location, walFileName)
	_, errO := os.Stat(oldPath)
	_, errN := os.Stat(newPath)

	if (errO == nil || !os.IsNotExist(errO)) || (errN == nil || !os.IsNotExist(errN)) {
		// Seems someone is doing something about this file
		return
	}

	err := os.MkdirAll(runningLocation, 0755)
	if err != nil {
		tracelog.ErrorLogger.Printf("WAL-prefetch %s, make dirs: %v", walFileName, err)
	}

	tracelog.DebugLogger.Printf("File prefetched to %s", oldPath)
	err = internal.DownloadFileTo(reader, walFileName, oldPath)
	if err != nil {
		tracelog.ErrorLogger.Printf("WAL-prefetch %s, download: %v", walFileName, err)
	} else {
		tracelog.DebugLogger.Printf("WAL-prefetch %s, download OK", walFileName)
	}

	_, errO = os.Stat(oldPath)
	_, errN = os.Stat(newPath)
	if errO == nil && os.IsNotExist(errN) {
		err = os.Rename(oldPath, newPath)
		if err != nil {
			tracelog.ErrorLogger.Printf("WAL-prefetch %s, rename %s -> %s: %v", walFileName, oldPath, newPath, err)
		}
	} else {
		_ = os.Remove(oldPath) // error is ignored
	}
}

func getPrefetchLocations(location string,
	walFileName string) (prefetchLocation string,
	runningLocation string,
	runningFile string,
	fetchedFile string) {
	if viper.IsSet(conf.PrefetchDir) {
		location = viper.GetString(conf.PrefetchDir)
	}
	prefetchLocation = path.Join(location, ".wal-g", "prefetch")
	runningLocation = path.Join(prefetchLocation, "running")
	oldPath := path.Join(runningLocation, walFileName)
	newPath := path.Join(prefetchLocation, walFileName)
	return prefetchLocation, runningLocation, oldPath, newPath
}
