// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dump

import (
	"encoding/csv"
	"os"
	"path"
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
)

type Dump struct {
	Options *DumpOptions
	DB      *database.Database
}

func (d *Dump) ExportMessages(destDir string, dur time.Duration) {
	now := time.Now()
	endTime := now.Add(-1 * dur)
	log.Infof("exporting messages before %s", endTime.String())

	dirPath := path.Join(destDir, "messages")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0777); err != nil {
			log.Error(err, "create dir error")
			return
		}
	}

	year, mon, _ := endTime.Date()
	msgFile, err := getDumpFile(dirPath, "messages", year, mon)
	if err != nil {
		log.Error(err, "get dump file message")
		return
	}
	defer msgFile.Close()
	alertMsgFile, err := getDumpFile(dirPath, "alert-messages", year, mon)
	if err != nil {
		log.Error(err, "get dump file alert-messages")
		return
	}
	defer alertMsgFile.Close()
	userMsgFile, err := getDumpFile(dirPath, "user-message-statuses", year, mon)
	if err != nil {
		log.Error(err, "get dump file user-message-statuses")
		return
	}
	defer userMsgFile.Close()

	msgWriter := csv.NewWriter(msgFile)
	msgWriter.Write((&models.Message{}).ColumnSlice())
	alertMsgWriter := csv.NewWriter(alertMsgFile)
	alertMsgWriter.Write((&models.AlertMessage{}).ColumnSlice())
	userMsgWriter := csv.NewWriter(userMsgFile)
	userMsgWriter.Write((&models.UserMessageStatus{}).ColumnSlice())

	msgCount := 0
	alertMsgCount := 0
	userMsgCount := 0
	for {
		// 避免message过多内存炸裂，每次导出100条
		msgs := []models.Message{}
		if err = d.DB.DB().
			Where("created_at < ?", endTime).
			Order("created_at").
			Limit(100).
			Find(&msgs).Error; err != nil {
			log.Error(err, "find messages")
			return
		}

		alertMsgs := []models.AlertMessage{}
		if err = d.DB.DB().
			Preload("AlertInfo"). // preload 以导出label
			Where("created_at < ?", endTime).
			Order("created_at").
			Limit(100).
			Find(&alertMsgs).Error; err != nil {
			log.Error(err, "find alert messages")
			return
		}

		if len(msgs) == 0 && len(alertMsgs) == 0 {
			log.Info("export messages to csv finished",
				"messages", msgCount,
				"alert-messages", alertMsgCount,
				"user-message-statuses", userMsgCount)
			return
		}

		// 写csv
		msgCsv := make([][]string, len(msgs))
		msgids := make([]uint, len(msgs))
		alertMsgCsv := make([][]string, len(alertMsgs))
		alertMsgids := make([]uint, len(alertMsgs))

		for i := range msgs {
			msgCsv[i] = msgs[i].ValueSlice()
			// 缓存id
			msgids[i] = msgs[i].ID
		}
		for i := range alertMsgs {
			alertMsgCsv[i] = alertMsgs[i].ValueSlice()
			// 缓存id
			alertMsgids[i] = alertMsgs[i].ID
		}
		usermsgs := []models.UserMessageStatus{}
		if err := d.DB.DB().Where("message_id in ? or alert_message_id in ?", msgids, alertMsgids).Find(&usermsgs).Error; err != nil {
			log.Error(err, "find user message")
			return
		}
		userMsgCsv := make([][]string, len(usermsgs))
		for i := range usermsgs {
			userMsgCsv[i] = usermsgs[i].ValueSlice()
		}

		if err := msgWriter.WriteAll(msgCsv); err != nil {
			log.Error(err, "write message csv")
			return
		}
		if err := alertMsgWriter.WriteAll(alertMsgCsv); err != nil {
			log.Error(err, "write alert message csv")
			return
		}
		if err := userMsgWriter.WriteAll(userMsgCsv); err != nil {
			log.Error(err, "write user message csv")
			return
		}

		// 删除数据
		if len(msgs) > 0 {
			if err := d.DB.DB().
				Where("id in ?", msgids).
				Delete(&models.Message{}).Error; err != nil {
				log.Error(err, "delete messages")
				return
			}
		}
		if len(alertMsgs) > 0 {
			if err := d.DB.DB().
				Where("id in ?", alertMsgids).
				Delete(&models.AlertMessage{}).Error; err != nil {
				log.Error(err, "delete alert messages")
				return
			}
		}
		if err := d.DB.DB().
			Where("message_id in ? or alert_message_id in ?", msgids, alertMsgids).
			Delete(&models.UserMessageStatus{}).Error; err != nil {
			log.Error(err, "delete user message")
			return
		}
		msgCount += len(msgs)
		alertMsgCount += len(alertMsgs)
		userMsgCount += len(usermsgs)
	}
}
