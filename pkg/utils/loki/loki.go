package loki

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func Exist(eles []interface{}, ele interface{}) bool {
	exist := false
	for _, element := range eles {
		if element.(map[string]interface{})["value"].(string) == ele.(string) {
			exist = true
			return exist
		}
	}
	return exist
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func GetLogLevelExpr(level string) string {
	switch level {
	case "info":
		return `(([[]I[]])|([[【= \t][iI](?i)nfo[]】 \t]))|(INFO)`
	case "debug":
		return `(([[]D[]])|([[【= \t][dD](?i)ebug[]】 \t]))|(DEBUG)`
	case "warn":
		return `(([[]W[]])|([[【= \t][wW](?i)arn((?i)(ing))?[]】 \t]))|(WARN)|(WARNING)`
	case "error":
		return `(([[]E[]])|([[【= \t][eE](?i)rror[]】 \t]))|(ERROR)`
	default:
		return `(([[]I[]])|([[【= \t][iI](?i)nfo[]】 \t]))|(INFO)|(([[]D[]])|([[【= \t][dD](?i)ebug[]】 \t]))|(DEBUG)|(([[]W[]])|([[【= \t][wW](?i)arn((?i)(ing))?[]】 \t]))|(WARN)|(WARNING)|(([[]E[]])|([[【= \t][eE](?i)rror[]】 \t]))|(ERROR)`
	}
}

func LogLevel(message string) string {
	reg := regexp.MustCompile(GetLogLevelExpr("info"))
	if reg.MatchString(message) {
		return "info"
	}
	reg = regexp.MustCompile(GetLogLevelExpr("debug"))
	if reg.MatchString(message) {
		return "debug"
	}
	reg = regexp.MustCompile(GetLogLevelExpr("warn"))
	if reg.MatchString(message) {
		return "warn"
	}
	reg = regexp.MustCompile(GetLogLevelExpr("error"))
	if reg.MatchString(message) {
		return "error"
	}
	return "unknown"
}

func GenerateLevelRegex(level string) string {
	all := GetLogLevelExpr("all")
	if strings.ToLower(level) == "unknown" {
		return fmt.Sprintf("!~ `%s`", all)
	}
	levelArray := []string{}
	levelUnknownExist := false
	levels := strings.Split(level, ",")
	levelExpr := ""
	for _, l := range levels {
		switch strings.ToLower(l) {
		case "info":
			levelArray = append(levelArray, GetLogLevelExpr("info"))
			all = strings.ReplaceAll(all, GetLogLevelExpr("info"), "")
		case "debug":
			levelArray = append(levelArray, GetLogLevelExpr("debug"))
			all = strings.ReplaceAll(all, GetLogLevelExpr("debug"), "")
		case "warn":
			levelArray = append(levelArray, GetLogLevelExpr("warn"))
			all = strings.ReplaceAll(all, GetLogLevelExpr("warn"), "")
		case "error":
			levelArray = append(levelArray, GetLogLevelExpr("error"))
			all = strings.ReplaceAll(all, GetLogLevelExpr("error"), "")
		case "unknown":
			levelUnknownExist = true
		}
	}
	if len(levelArray) > 0 {
		levelExpr = fmt.Sprintf("|~ `%s`", strings.Join(levelArray, "|"))
	}
	if levelUnknownExist {
		all = strings.Trim(all, "|")
		if all == "" {
			return ""
		}
		levelExpr = fmt.Sprintf("!~ `%s`", all)
	}

	return levelExpr
}

func SplitDateTime(start string, end string, limit int) ([]int, int) {
	// 切20份  1586330540000 000000
	startIndex, _ := strconv.Atoi(start[0:13])
	endIndex, _ := strconv.Atoi(end[0:13])

	step := (endIndex - startIndex) / limit
	index := 0
	splitDateTimeArray := []int{}
	for {
		if index < limit {
			splitDateTimeArray = append(splitDateTimeArray, startIndex+step*index)
			index++
			continue
		}
		break
	}

	return splitDateTimeArray, step
}

func InitSplitDateTime(limit int) map[string][]int {
	chartData := make(map[string][]int)
	chartData["info"] = make([]int, limit)
	chartData["debug"] = make([]int, limit)
	chartData["warn"] = make([]int, limit)
	chartData["error"] = make([]int, limit)
	chartData["unknown"] = make([]int, limit)
	return chartData
}

func TimeInPart(splitDateTime []int, timestamp string, step int) int {
	timestampIndex, _ := strconv.Atoi(timestamp[0:13])
	stepSum := (timestampIndex - splitDateTime[0]) / step

	return stepSum
}

func ShellHighlightShow(message string) string {
	highlight := message
	regExceptFilter, _ := regexp.Compile(`[[]\d*;?\d*m.*?[[]\d*;?\d*m`)
	if regExceptFilter.MatchString(message) {
		regInnerFilter, _ := regexp.Compile(`[[](\d*);?(\d*)m(.*)[[]\d*;?\d*m`)
		highlight = regExceptFilter.ReplaceAllStringFunc(highlight, func(item string) string {
			strs := regInnerFilter.FindStringSubmatch(item)
			if len(strs) == 4 {
				color := GetShellColor(strs[1])
				if color == "" {
					color = GetShellColor(strs[2])
				}
				if strs[1] == "1" || strs[1] == "" {
					return fmt.Sprintf(`<b style="color: %s !important;">%s</b>`, color, strs[3])
				} else {
					return fmt.Sprintf(`<b style="background-color: %s !important;color: slategray;">%s</b>`, color, strs[3])
				}
			}
			return item
		})
	}
	return highlight
}

func RegexHighlightShow(message string, filter string) string {
	highlight := message
	if filter != "" {
		regFilter, _ := regexp.Compile(filter)
		highlight = regFilter.ReplaceAllStringFunc(highlight, func(item string) string {
			return fmt.Sprintf(`<b style="color: #fb8c00 !important;">%s</b>`, item)
		})

	}
	return highlight
}

func GetShellColor(colorNo string) string {
	switch colorNo {
	case "40", "30":
		return "black"
	case "41", "31":
		return "red"
	case "42", "32":
		return "lightgreen"
	case "43", "33":
		return "yellow"
	case "44", "34":
		return "blue"
	case "45", "35":
		return "magenta"
	case "46", "36":
		return "cyan"
	case "47", "37":
		return "wheat"
	}
	return ""
}

func GetExpr(label string, value string) string {
	queryExpr := ""
	if label != "" && value != "" {
		queryExpr = fmt.Sprintf("%s=\"%s\"", label, value)
	}
	return queryExpr
}
