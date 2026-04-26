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

// ParseSubtitle 自动检测并解析字幕，并对超长条目进行智能分段
func ParseSubtitle(data []byte) ([]model.SubtitleEntry, error) {
	content := string(data)
	var entries []model.SubtitleEntry
	var err error

	if strings.HasPrefix(strings.TrimSpace(content), "WEBVTT") {
		entries, err = ParseVTT(data)
	} else {
		entries, err = ParseSRT(data)
	}
	if err != nil {
		return nil, err
	}

	entries = SegmentSubtitles(entries)
	return entries, nil
}

// SegmentSubtitles 对超长字幕条目进行智能分段
// ASR 生成的字幕通常只有 1-2 条，时间跨度覆盖整个视频
// 此函数先尝试按标点符号切分，若无标点则按固定长度切分
func SegmentSubtitles(entries []model.SubtitleEntry) []model.SubtitleEntry {
	var result []model.SubtitleEntry
	idx := 0

	for _, entry := range entries {
		duration := entry.EndTime - entry.StartTime
		if duration <= 8 || len([]rune(entry.Text)) <= 30 {
			idx++
			entry.Index = idx
			entry.Text = cleanASRText(entry.Text)
			result = append(result, entry)
			continue
		}

		segments := splitByPunctuation(entry.Text)
		if len(segments) <= 1 {
			segments = splitByLength(entry.Text, 25)
		}
		if len(segments) <= 1 {
			idx++
			entry.Index = idx
			entry.Text = cleanASRText(entry.Text)
			result = append(result, entry)
			continue
		}

		totalRunes := 0
		for _, seg := range segments {
			totalRunes += len([]rune(seg))
		}

		currentTime := entry.StartTime
		for _, seg := range segments {
			segRunes := len([]rune(seg))
			segDuration := duration * float64(segRunes) / float64(totalRunes)
			endTime := currentTime + segDuration
			if endTime > entry.EndTime {
				endTime = entry.EndTime
			}

			idx++
			result = append(result, model.SubtitleEntry{
				Index:     idx,
				StartTime: currentTime,
				EndTime:   endTime,
				Text:      cleanASRText(seg),
			})
			currentTime = endTime
		}
	}

	return result
}

// splitByPunctuation 按标点符号将长文本切分为短句
func splitByPunctuation(text string) []string {
	sentenceEnders := []string{
		"。",
		"！",
		"？",
		"；",
		".",
		"!",
		"?",
		";",
		"…",
		"……",
		"\n",
	}

	type splitPos struct {
		start int
		end   int
	}

	textRunes := []rune(text)
	var positions []splitPos
	lastEnd := 0

	for i := 0; i < len(textRunes); i++ {
		matched := false
		for _, ender := range sentenceEnders {
			enderRunes := []rune(ender)
			if i+len(enderRunes) <= len(textRunes) {
				match := true
				for k := 0; k < len(enderRunes); k++ {
					if textRunes[i+k] != enderRunes[k] {
						match = false
						break
					}
				}
				if match {
					cutEnd := i + len(enderRunes)
					if cutEnd > lastEnd {
						segment := strings.TrimSpace(string(textRunes[lastEnd:cutEnd]))
						if segment != "" {
							positions = append(positions, splitPos{lastEnd, cutEnd})
						}
						lastEnd = cutEnd
					}
					i += len(enderRunes) - 1
					matched = true
					break
				}
			}
		}

		if !matched && i == len(textRunes)-1 && lastEnd < len(textRunes) {
			segment := strings.TrimSpace(string(textRunes[lastEnd:]))
			if segment != "" {
				positions = append(positions, splitPos{lastEnd, len(textRunes)})
			}
		}
	}

	if len(positions) == 0 {
		return []string{text}
	}

	var segments []string
	for _, pos := range positions {
		seg := strings.TrimSpace(string(textRunes[pos.start:pos.end]))
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	var merged []string
	for _, seg := range segments {
		if len(merged) > 0 && len([]rune(merged[len(merged)-1]))+len([]rune(seg)) < 15 {
			merged[len(merged)-1] = merged[len(merged)-1] + seg
		} else {
			merged = append(merged, seg)
		}
	}

	return merged
}

// cleanASRText 清理 ASR 生成字幕中的多余空格
// ASR 字幕特征：每个字之间有空格（如 "这 是 一 个 测 试"）
// 检测到这种模式时去掉字间空格
func cleanASRText(text string) string {
	text = strings.TrimSpace(text)

	chars := []rune(text)
	spaceCount := 0
	nonSpaceCount := 0
	for _, c := range chars {
		if c == ' ' {
			spaceCount++
		} else {
			nonSpaceCount++
		}
	}

	if nonSpaceCount > 0 && float64(spaceCount)/float64(nonSpaceCount) > 0.3 {
		var buf []rune
		for i, c := range chars {
			if c == ' ' {
				nextNonSpace := false
				for j := i + 1; j < len(chars); j++ {
					if chars[j] != ' ' {
						if chars[j] >= 0x4E00 && chars[j] <= 0x9FFF {
							nextNonSpace = true
						}
						break
					}
				}
				prevNonSpace := false
				if i > 0 {
					for j := i - 1; j >= 0; j-- {
						if chars[j] != ' ' {
							if chars[j] >= 0x4E00 && chars[j] <= 0x9FFF {
								prevNonSpace = true
							}
							break
						}
					}
				}
				if prevNonSpace && nextNonSpace {
					continue
				}
			}
			buf = append(buf, c)
		}
		text = string(buf)
	}

	text = strings.ReplaceAll(text, "  ", " ")
	text = strings.TrimSpace(text)
	return text
}

// splitByLength 按固定字符数切分无标点的长文本
// 优先在空格处断句，避免切断词语
func splitByLength(text string, maxRunes int) []string {
	textRunes := []rune(text)
	if len(textRunes) <= maxRunes {
		return []string{text}
	}

	var segments []string
	start := 0

	for start < len(textRunes) {
		end := start + maxRunes
		if end >= len(textRunes) {
			seg := strings.TrimSpace(string(textRunes[start:]))
			if seg != "" {
				segments = append(segments, seg)
			}
			break
		}

		bestBreak := end
		spaceRange := 8
		searchStart := end - spaceRange
		if searchStart < start {
			searchStart = start
		}
		for i := end; i >= searchStart; i-- {
			if i < len(textRunes) && textRunes[i] == ' ' {
				bestBreak = i + 1
				break
			}
		}

		seg := strings.TrimSpace(string(textRunes[start:bestBreak]))
		if seg != "" {
			segments = append(segments, seg)
		}
		start = bestBreak
	}

	return segments
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
