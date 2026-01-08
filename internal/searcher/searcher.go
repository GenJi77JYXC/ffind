package searcher

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Config struct {
	StartDir    string
	Keyword     string
	IgnoreCase  bool
	Exts        []string
	ExcludeDirs []string
	Workers     int // 新增：并发工作者数量
	Regexp      bool
}

func Search(cfg Config) (matchCount int, fileCount int, duration time.Duration, err error) {
	start := time.Now()
	// 设置默认并发数
	if cfg.Workers <= 0 {
		cfg.Workers = 4 // 默认 4 个 worker
	}

	keyword := cfg.Keyword
	if cfg.IgnoreCase {
		keyword = strings.ToLower(keyword)
	}

	// 通道：用于传递待处理的文件路径
	fileChan := make(chan string, 100)

	// 结果统计（线程安全）
	var mu sync.Mutex
	matchCount = 0
	fileCount = 0

	// 启动 worker goroutine
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				processFile(path, cfg, &mu, &matchCount, &fileCount)
			}
		}()
	}

	// 遍历目录，收集文件路径（这个过程保持单线程，避免竞争）
	err = filepath.Walk(cfg.StartDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的路径
		}

		// 排除指定目录
		if info.IsDir() {
			base := filepath.Base(path)
			for _, excl := range cfg.ExcludeDirs {
				if base == excl {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// 扩展名过滤
		if len(cfg.Exts) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			allowed := false
			for _, e := range cfg.Exts {
				if ext == "."+strings.ToLower(e) {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil
			}
		}

		// 只处理普通文件
		if info.Mode().IsRegular() {
			fileChan <- path
		}
		return nil
	})

	close(fileChan) // 所有文件收集完毕
	wg.Wait()       // 等待所有 worker 完成

	if err != nil {
		return 0, 0, 0, err
	}

	duration = time.Since(start)
	return matchCount, fileCount, duration, nil
}

// processFile 处理单个文件
func processFile(path string, cfg Config, mu *sync.Mutex, matchCount, fileCount *int) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	fileMatched := false

	var re *regexp.Regexp
	if cfg.Regexp {
		pattern := cfg.Keyword
		if cfg.IgnoreCase {
			pattern = "(?i)" + pattern
		}
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			// 正则无效，跳过文件
			return
		}
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var matches []int // 匹配的起始和结束索引（成对）

		if cfg.Regexp {
			if re == nil {
				continue
			}
			matches = re.FindStringIndex(line)
			if matches == nil {
				continue
			}
			// 支持一行多个匹配
			allMatches := re.FindAllStringIndex(line, -1)
			if len(allMatches) == 0 {
				continue
			}

			mu.Lock()
			if !fileMatched {
				color.Cyan("\n%s:", path)
				*fileCount++
				fileMatched = true
			}
			*matchCount += len(allMatches) // 每个匹配算一个
			mu.Unlock()

			color.Yellow("  %d: ", lineNum)

			// 高亮所有匹配部分
			pos := 0
			for _, m := range allMatches {
				start, end := m[0], m[1]
				// 打印未匹配部分
				fmt.Print("  " + line[pos:start])
				// 高亮匹配部分
				color.New(color.FgYellow, color.Bold).Print(line[start:end])
				pos = end
			}
			// 打印剩余部分
			if pos < len(line) {
				fmt.Println("  " + line[pos:])
			} else {
				fmt.Println()
			}

		} else {
			// 普通关键词模式（保持你之前的高亮逻辑，推荐用这个简化版）
			searchLine := line
			searchKeyword := cfg.Keyword
			if cfg.IgnoreCase {
				searchLine = strings.ToLower(searchLine)
				searchKeyword = strings.ToLower(searchKeyword)
			}

			if !strings.Contains(searchLine, searchKeyword) {
				continue
			}

			mu.Lock()
			if !fileMatched {
				color.Cyan("\n%s:", path)
				*fileCount++
				fileMatched = true
			}
			*matchCount++
			mu.Unlock()

			color.Yellow("  %d: ", lineNum)

			// 普通模式高亮（支持忽略大小写）
			lowerLine := strings.ToLower(line)
			lowerKeyword := strings.ToLower(cfg.Keyword)
			start := 0
			for {
				idx := strings.Index(lowerLine[start:], lowerKeyword)
				if idx == -1 {
					break
				}
				absIdx := start + idx
				fmt.Print("  " + line[start:absIdx])
				color.New(color.FgYellow, color.Bold).Print(line[absIdx : absIdx+len(cfg.Keyword)])
				start = absIdx + len(cfg.Keyword)
			}
			if start < len(line) {
				fmt.Println("  " + line[start:])
			} else {
				fmt.Println()
			}
		}
	}
}
