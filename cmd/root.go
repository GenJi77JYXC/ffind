package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"www.genji.xin/backend/ffind/internal/searcher"
)

const (
	version = "v1.0.0"
	author  = "GenJi (@GenJi_JYXC)"
)

var (
	dir         string
	keyword     string
	ignoreCase  bool
	exts        []string
	excludeDirs []string
	workers     int
	useRegexp   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "ffind [keyword] [path...]",
	Version: version,
	Short:   "快速在文件中搜索关键词",
	Long: `ffind 是一个用 Go 写的快速文件内容搜索工具。
支持并发、忽略大小写、指定扩展名等功能。`,
	Args: cobra.MinimumNArgs(1), // 至少 1 个参数：关键词，可跟路径
	Run: func(cmd *cobra.Command, args []string) {
		// 第一个参数是关键词
		keyword := args[0]

		// 剩余参数是搜索路径，没有则默认当前目录
		var searchPaths []string
		if len(args) > 1 {
			searchPaths = args[1:]
		} else {
			searchPaths = []string{"."}
		}

		var totalMatches, totalFiles int
		var totalDuration time.Duration
		first := true

		for _, path := range searchPaths {
			// 检查路径是否存在
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("路径不存在: %s\n", path)
					continue
				}
			}

			// 如果不是第一个路径，打印空行分隔
			if !first {
				fmt.Println()
			}
			first = false

			fmt.Printf("搜索路径: %s\n", path)

			cfg := searcher.Config{
				StartDir:    path,
				Keyword:     keyword,
				IgnoreCase:  ignoreCase,
				Exts:        exts,
				ExcludeDirs: excludeDirs,
				Workers:     workers,
				Regexp:      useRegexp,
			}

			matches, files, duration, err := searcher.Search(cfg)
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				continue
			}

			totalMatches += matches
			totalFiles += files
			totalDuration += duration

			// 每个路径单独打印小结
			fmt.Printf("→ 本路径: %d 个匹配项，%d 个文件，耗时 %v\n", matches, files, duration)
		}

		// 多路径时打印总计
		if len(searchPaths) > 1 {
			fmt.Printf("\n=== 总计 ===\n")
			fmt.Printf("找到 %d 个匹配项，分布在 %d 个文件中。\n", totalMatches, totalFiles)
			fmt.Printf("总搜索耗时: %v\n", totalDuration)
		} else if len(searchPaths) == 1 {
			// 单个路径时，也打印一个简洁总计
			fmt.Printf("\n找到 %d 个匹配项，分布在 %d 个文件中。\n", totalMatches, totalFiles)
			fmt.Printf("搜索耗时: %v\n", totalDuration)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 定义所有 flag
	rootCmd.Flags().StringVarP(&dir, "dir", "d", ".", "搜索起始目录")
	rootCmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "忽略大小写")
	rootCmd.Flags().StringSliceVarP(&exts, "ext", "e", []string{}, "只搜索指定扩展名（如 go,md,txt）")
	rootCmd.Flags().StringSliceVar(&excludeDirs, "exclude-dir", []string{".git", "node_modules", "vendor"}, "排除目录")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 0, "并发工作者数量（0=自动）")
	rootCmd.Flags().BoolVarP(&useRegexp, "regexp", "r", false, "使用正则表达式搜索")
	// 添加 -v 作为 --version 的缩写
	rootCmd.Flags().BoolP("version", "v", false, "显示版本信息")
	rootCmd.SetVersionTemplate(`ffind {{.Version}}
{{.Long}}
作者: ` + author + `
GitHub: 
`)

}
