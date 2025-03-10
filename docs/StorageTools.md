## Storage tools (danger zone)
`wal-g st` command series allows interacting with the configured storage. Be aware that these commands can do potentially harmful operations and make sure that you know what you're doing.

### ``ls``
Prints listing of the objects in the provided storage folder.

``wal-g st ls`` get listing with all objects and folders in the configured storage.

``wal-g st ls -r`` get recursive listing with all objects in the configured storage.

``wal-g st ls some_folder/some_subfolder`` get listing with all objects in the provided storage path.

### ``get``
Download the specified storage object. By default, the command will try to apply the decompression and decryption (if configured).

Flags:

1. Add `--no-decompress` to download the remote object without decompression
2. Add `--no-decrypt` to download the remote object without decryption

Examples:

``wal-g st get path/to/remote_file path/to/local_file`` download the file from storage.

``wal-g st get path/to/remote_file path/to/local_file --no-decrypt`` download the file from storage without decryption.

### ``cat``
Show the specified storage object to STDOUT. 
By default, the command will NOT try to decompress and decrypt it.
Useful for getting sentinels and other meta-information files.

Flags:

1. Add `--decompress` to decompress source file
2. Add `--decrypt` to decrypt source file

Examples:

``wal-g st cat path/to/remote_file.json`` show `remote_file.json`

### ``rm``
Remove the specified storage object(s). 
Any prefix may be specified as the argument. If there's a file with this path, it is removed. If not, but there's a directory with this path - all files from it and its subdirectories are removed.

Examples:

``wal-g st rm path/to/remote_file`` remove the file from storage.

``wal-g st rm path/to/remote_file_or_directory`` remove a file or all files in the directory.

``wal-g st rm path/to/remote_directory/`` explicitly specify that the path points to a directory, not a file.

### ``put``
Upload the specified file to the storage. By default, the command will try to apply the compression and encryption (if configured).

Flags:

1. Add `--no-compress` to upload the object without compression
2. Add `--no-encrypt` to upload the object without encryption

Example:

``wal-g st put path/to/local_file path/to/remote_file`` upload the local file to the storage.


### ``get-stream``
Download the specified backup as single stream (when backup is stream-based backup). This command will:
* decrypt backup
* decompress backup
* assemble parts of backups (when `WALG_STREAM_SPLITTER_*` used)
* save to file

Example:

``wal-g st get-stream BACKUP_NAME path/to/local_file`` upload the local file to the storage.

``wal-g st get-stream BACKUP_NAME | xbstream --extract`` upload the local file to the storage.

### `transfer`
Transfer files from one configured storage to another. Is usually used to move files from a failover storage to the primary one when it becomes alive.

Subcommands:
1. `transfer files prefix` - moves arbitrary files without any special treatment.
   
   Argument `prefix` is path to a directory in both storages, where files should be moved to/from. Files from all subdirectories are also moved.

2. `transfer pg-wals` - moves PostgreSQL WAL files only (just an alias for `transfer files "wal_005/"`).

3. `transfer backups [--max-backups=N]` - consistently moves backups.

   To prevent any problems with restoring from a partially uploaded/removed backup, the signal file `*_backup_stop_sentinel.json` is moved to the source storage last, and deleted from the target storage first.

   An additional flag is supported: `--max-backups` specifies max number of backups to move in this run.

Flags (supported in every subcommand):

1. Add `-s (--source)` to specify the source storage name to take files from. To specify the primary storage, use `default`. This flag is required.

2. Add `-t (--target)` to specify the target storage name to save files to. The primary storage is used by default.

3. Add `-o (--overwrite)` to move files and overwrite them, even if they already existed in the target storage.

   Files existing in both storages will remain as they are if this flag isn't specified.

   Please note that the files are checked for their existence in the target storage only once at the very beginning. So if a new file appear in the target storage while the command is working, it may be overwritten even when `--overwrite` isn't specified.

4. Add `--fail-fast` so that the command stops after the first error occurs with transferring any file. 

   Without this flag the command will try to move every file.

   Regardless of the flag, the command will end with zero error code only if all the files have moved successfully.

   Keep in mind that files aren't transferred atomically. This means that when this flag is set, an error occured with one file may interrupt transferring other files in the middle, so they may already be copied to the target storage, but not yet deleted from the source. 

5. Add `-c (--concurrency)` to set the max number of concurrent workers that will move files.

6. Add `-m (--max-files)` to set the max number of files to move in a single command run.

7. Add `--appearance-checks` to set the max number of checks for files to appear in the target storage, which will be performed after moving the file and before deleting it.

   This option is recommended for use with storages that don't guarantee the read-after-write consistency. 
   Otherwise, transferring files between them may cause a moment of time, when a file doesn't exist in both storages, which may lead to problems with restoring backups at that moment.

8. Add `--appearance-checks-interval` to specify the min time interval between checks of the same file to appear in the target storage.

   The duration must be specified in the golang `time.Duration` [format](https://pkg.go.dev/time#ParseDuration).

9. Add `--preserve` to prevent transferred files from being deleted from the source storage ("copy" files instead of "moving").

Examples:

``wal-g st transfer pg-wals --source='my_failover_ssh'``

``wal-g st transfer files folder/single_file.json --source='default' --target='my_failover_ssh' --overwrite``

``wal-g st transfer files basebackups_005/ --source='my_failover_s3' --target='default' --fail-fast -c=50 -m=10000 --appearance-checks=5 --appearance-checks-interval=1s``

``wal-g st transfer backups --source='my_failover_s3' --target='default' --fail-fast -c=50 --max-files=10000 --max-backups=10 --appearance-checks=5 --appearance-checks-interval=1s``
