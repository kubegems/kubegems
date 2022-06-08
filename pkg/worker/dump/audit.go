package dump

import (
	"encoding/csv"
	"os"
	"path"
	"strconv"
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
)

func (d *Dump) ExportAuditlogs(destDir string, dur time.Duration) {
	now := time.Now()
	endTime := now.Add(-1 * dur)
	log.Infof("exporting auditlogs before %s", endTime.String())

	dirPath := path.Join(destDir, "audit")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0777); err != nil {
			log.Error(err, "create dir error")
			return
		}
	}

	// 使用截止当月作为文件名，保证同一月的数据写入同一个文件
	year, mon, _ := endTime.Date()
	file, err := getDumpFile(dirPath, "auditlogs", year, mon)
	if err != nil {
		log.Error(err, "get dump file error")
		return
	}
	defer file.Close()

	w := csv.NewWriter(file)
	w.Write([]string{"id", "user_name", "tenant", "module", "action", "success", "raw_data", "labels", "client_ip", "name", "created_at", "updated_at", "deleted_at"})

	count := 0
	for {
		// 避免AuditLog过多内存炸裂，每次导出100条
		auditlogs := []models.AuditLog{}
		if err = d.DB.DB().
			Unscoped(). // 有delete_at 字段
			Where("created_at < ?", endTime).
			Order("created_at").
			Limit(100).
			Find(&auditlogs).Error; err != nil {
			log.Error(err, "find auditlogs error")
			return
		}

		if len(auditlogs) == 0 {
			log.Info("export auditlogs to csv finished", "total", count)
			return
		}

		// 写csv
		data := make([][]string, len(auditlogs))
		ids := make([]uint, len(auditlogs))
		for i := range auditlogs {
			data[i] = []string{
				strconv.Itoa(int(auditlogs[i].ID)),
				auditlogs[i].Username,
				auditlogs[i].Tenant,
				auditlogs[i].Module,
				auditlogs[i].Action,
				utils.BoolToString(auditlogs[i].Success),
				auditlogs[i].RawData.String(),
				auditlogs[i].Labels.String(),
				auditlogs[i].ClientIP,
				auditlogs[i].Name,
				auditlogs[i].CreatedAt.Format("2006-01-02 15:04:05.000"),      // mysql datetime 格式
				auditlogs[i].UpdatedAt.Format("2006-01-02 15:04:05.000"),      // mysql datetime 格式
				auditlogs[i].DeletedAt.Time.Format("2006-01-02 15:04:05.000"), // mysql datetime 格式
			}
			ids[i] = auditlogs[i].ID
		}
		if err := w.WriteAll(data); err != nil {
			log.Error(err, "write auditlogs to csv")
			return
		}

		// 删除数据
		if err := d.DB.DB().
			Unscoped(). // 有delete_at 字段，永久删除
			Where("id in ?", ids).
			Delete(&models.AuditLog{}).Error; err != nil {
			log.Error(err, "delete auditlogs")
			return
		}
		count += len(auditlogs)
	}
}
