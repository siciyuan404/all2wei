package service

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"all2wei/internal/model"
)

var (
	srtTimePattern = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})`)
)

// ParseSRT 解析 SRT 格式字幕
func ParseSRT(data []byte) ([]model.SubtitleEntry, error) {
	var entries []model.SubtitleEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	
	var currentEntry model.SubtitleEntry
	var inTextBlock bool
	var textLines []string
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 空行表示一个条目结束
		if line == "" {
			if currentEntry.Index > 0 && len(textLines) > 0 {
				currentEntry.Text = strings.Join(textLines, "\n")
				entries = append(entries, currentEntry)
			}
			currentEntry = model.SubtitleEntry{}
			textLines = nil
			inTextBlock = false
			continue
		}
		
		// 解析序号
		if !inTextBlock && currentEntry.Index == 0 {
			if idx, err := strconv.Atoi(line); err == nil {
				currentEntry.Index = idx
				continue
			}
		}
		
		// 解析时间轴
		if strings.Contains(line, "-->") {
			times := strings.Split(line, "-->")
			if len(times) == 2 {
				start, err1 := parseSRTTime(strings.TrimSpace(times[0]))
				end, err2 := parseSRTTime(strings.TrimSpace(times[1]))
				if err1 == nil && err2 == nil {
					currentEntry.StartTime = start
					currentEntry.EndTime = end
					inTextBlock = true
					continue
				}
			}
		}
		
		// 文本内容
		if inTextBlock || currentEntry.StartTime > 0 {
			textLines = append(textLines, line)
		}
	}
	
	// 处理最后一个条目
	if currentEntry.Index > 0 && len(textLines) > 0 {
		currentEntry.Text = strings.Join(textLines, "\n")
		entries = append(entries, currentEntry)
	}
	
	return entries, scanner.Err()
}

// ParseVTT 解析 WebVTT 格式字幕
func ParseVTT(data []byte) ([]model.SubtitleEntry, error) {
	var entries []model.SubtitleEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	
	var index int
	var inTextBlock bool
	var textLines []string
	var currentEntry model.SubtitleEntry
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过 WEBVTT 头
		if strings.HasPrefix(line, "WEBVTT") {
			continue
		}
		
		// 空行表示条目结束
		if line == "" {
			if inTextBlock && len(textLines) > 0 {
				currentEntry.Text = strings.Join(textLines, "\n")
				entries = append(entries, currentEntry)
			}
			inTextBlock = false
			textLines = nil
			currentEntry = model.SubtitleEntry{}
			continue
		}
		
		// 解析时间轴
		if strings.Contains(line, "-->") {
			index++
			currentEntry.Index = index
			
			times := strings.Split(line, "-->")
			if len(times) == 2 {
				start, err1 := parseVTTTime(strings.TrimSpace(times[0]))
				end, err2 := parseVTTTime(strings.TrimSpace(times[1]))
				if err1 == nil && err2 == nil {
					currentEntry.StartTime = start
					currentEntry.EndTime = end
					inTextBlock = true
					continue
				}
			}
		}
		
		// 文本内容
		if inTextBlock {
			textLines = append(textLines, line)
		}
	}
	
	// 处理最后一个条目
	if inTextBlock && len(textLines) > 0 {
		currentEntry.Text = strings.Join(textLines, "\n")
		entries = append(entries, currentEntry)
	}
	
	return entries, scanner.Err()
}

// ParseSubtitle 自动检测并解析字幕
func ParseSubtitle(data []byte) ([]model.SubtitleEntry, error) {
	content := string(data)
	if strings.HasPrefix(strings.TrimSpace(content), "WEBVTT") {
		return ParseVTT(data)
	}
	return ParseSRT(data)
}

func parseSRTTime(timeStr string) (float64, error) {
	matches := srtTimePattern.FindStringSubmatch(timeStr)
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	
	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	millis, _ := strconv.Atoi(matches[4])
	
	d := time.Duration(hours)*time.Hour + 
		time.Duration(minutes)*time.Minute + 
		time.Duration(seconds)*time.Second + 
		time.Duration(millis)*time.Millisecond
	
	return d.Seconds(), nil
}

func parseVTTTime(timeStr string) (float64, error) {
	// VTT 格式: 00:00:00.000 或 00:00.000
	timeStr = strings.TrimSpace(timeStr)
	parts := strings.Split(timeStr, ":")
	
	var hours, minutes int
	var seconds float64
	
	if len(parts) == 3 {
		hours, _ = strconv.Atoi(parts[0])
		minutes, _ = strconv.Atoi(parts[1])
		seconds, _ = strconv.ParseFloat(parts[2], 64)
	} else if len(parts) == 2 {
		minutes, _ = strconv.Atoi(parts[0])
		seconds, _ = strconv.ParseFloat(parts[1], 64)
	} else {
		return 0, fmt.Errorf("invalid vtt time format: %s", timeStr)
	}
	
	d := time.Duration(hours)*time.Hour + 
		time.Duration(minutes)*time.Minute + 
		time.Duration(seconds*1000)*time.Millisecond
	
	return d.Seconds(), nil
}
