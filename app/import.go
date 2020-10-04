package app

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) SlackImport(fileData multipart.File, fileSize int64, teamID string) (*model.AppError, *bytes.Buffer) {
	log := bytes.NewBufferString(utils.T("api.slackimport.slack_import.log"))

	zipreader, err := zip.NewReader(fileData, fileSize)
	if err != nil || zipreader.File == nil {
		log.WriteString(utils.T("api.slackimport.slack_import.zip.app_error"))
		return model.NewAppError("SlackImport", "api.slackimport.slack_import.zip.app_error", nil, err.Error(), http.StatusBadRequest), log
	}

	file := zipreader.File[0]
	if file.UncompressedSize64 > slackImportMaxFileSize {
		log.WriteString(utils.T("api.slackimport.slack_import.zip.file_too_large", map[string]interface{}{"Filename": file.Name}))
		return
	}

	reader, err := file.Open()

	a.BulkImport(reader, false, 3)
}

// データアップロード
func (a *App) BulkImport(fileReader io.Reader, dryRun bool, workers int) (*model.AppError, int) {
	scanner := bufio.NewScanner(fileReader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNumber := 0

	//a.Srv().Store.LockToMaster()
	//defer a.Srv().Store.UnlockFromMaster()

	errorsChan := make(chan LineImportWorkerError, (2*workers)+1) // size chosen to ensure it never gets filled up completely.
	var wg sync.WaitGroup
	var linesChan chan LineImportWorkerData
	lastLineType := ""

	for scanner.Scan() {
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		lineNumber++

		var line LineImportData
		if err := decoder.Decode(&line); err != nil {
			return model.NewAppError("BulkImport", "app.import.bulk_import.json_decode.error", nil, err.Error(), http.StatusBadRequest), lineNumber
		}

		if lineNumber == 1 {
			importDataFileVersion, appErr := processImportDataFileVersionLine(line)
			if appErr != nil {
				return appErr, lineNumber
			}

			if importDataFileVersion != 1 {
				return model.NewAppError("BulkImport", "app.import.bulk_import.unsupported_version.error", nil, "", http.StatusBadRequest), lineNumber
			}
			lastLineType = line.Type
			continue
		}

		if line.Type != lastLineType {
			// ここでif文+コメント書くのは勿体無い..
			// Only clear the worker queue if is not the first data entry
			if lineNumber != 2 {
				// Changing type. Clear out the worker queue before continuing.
				// これによりworker側のforループが止まり、wg.Done()が呼ばれる。
				// 前回のTypeの処理をやめて、
				close(linesChan)
				wg.Wait()

				// Check no errors occurred while waiting for the queue to empty.
				if len(errorsChan) != 0 {
					err := <-errorsChan
					if stopOnError(err) {
						return err.Error, err.LineNumber
					}
				}
			}
			// Set up the workers and channel for this type.
			lastLineType = line.Type
			// 別のTypeを処理するループを開始しておく。
			linesChan = make(chan LineImportWorkerData, workers)
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go a.bulkImportWorker(dryRun, &wg, linesChan, errorsChan)
			}
		}

		select { // 1行の処理をループ側に任せるため。
		// なお、1メッセージはランダムな1つだけのgoルーチンによって処理される。
		// 複数のループが同一のメッセージを同時処理することは無い。
		case linesChan <- LineImportWorkerData{line, lineNumber}:
		// これはどうなる？
		case err := <-errorsChan:
			if stopOnError(err) {
				close(linesChan)
				wg.Wait()
				return err.Error, err.LineNumber
			}
		}
	}

	// No more lines. Clear out the worker queue before continuing.
	if linesChan != nil {
		close(linesChan)
	}
	wg.Wait()

	// Check no errors occurred while waiting for the queue to empty.
	if len(errorsChan) != 0 {
		err := <-errorsChan
		if stopOnError(err) {
			return err.Error, err.LineNumber
		}
	}

	if err := scanner.Err(); err != nil {
		return model.NewAppError("BulkImport", "app.import.bulk_import.file_scan.error", nil, err.Error(), http.StatusInternalServerError), 0
	}

	return nil, 0
}
