package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/treeforest/gohotdeploy/config"
	_ "github.com/treeforest/gohotdeploy/statik"
	"github.com/treeforest/gohotdeploy/webhook"

	"github.com/gookit/goutil/fsutil"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog/log"
	"github.com/treeforest/shell"
	"github.com/treeforest/tarutil"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	cfgPath := flag.String("config", "config.yml", "specify config yml path")
	flag.Parse()

	serverConfig, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	waitGroup, ctx := errgroup.WithContext(ctx)

	runHTTPServer(ctx, waitGroup, serverConfig)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("error from wait group")
	}
}

func runHTTPServer(ctx context.Context, waitGroup *errgroup.Group, serverConfig *config.ServerConfig) {
	ctrl := NewController(serverConfig)

	mux := &http.ServeMux{}
	mux.HandleFunc("/", webhook.Handler(ctrl.Dispatch(ctx, waitGroup)))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", serverConfig.Port),
		Handler: mux,
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("start HTTP webhook server at %s", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("HTTP webhook server failed to serve")
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown HTTP webhook server")

		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown HTTP webhook server")
			return err
		}

		log.Info().Msg("HTTP webhook server is stopped")
		return nil
	})
}

// Controller 是一个热部署控制器，主要负责将接收到的 GitLab 的 Webhook 事件分发到对应的仓库处理通道（Channel）。
type Controller struct {
	mutex      sync.Mutex           // 互斥锁，用于保护 channelMap 的并发访问
	channelMap map[string]*Channel  // 存储仓库处理通道的映射关系
	statikFS   http.FileSystem      // 存储静态文件系统，用于加载静态资源
	conf       *config.ServerConfig // 配置信息
}

func NewController(conf *config.ServerConfig) *Controller {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load file system")
	}
	return &Controller{
		channelMap: make(map[string]*Channel),
		statikFS:   statikFS,
		conf:       conf,
	}
}

// Dispatch 返回一个用于事件分发的函数。它的功能是接收到 Webhook 事件后，根据仓库名称找到对应的仓库处理通道（Channel），
// 然后将 Webhook 投递到通道并等待处理。
func (c *Controller) Dispatch(ctx context.Context, waitGroup *errgroup.Group) webhook.DispatchFunc {
	return func(hook *webhook.Webhook) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		channelName := hook.Repository.Name

		repo, ok := c.conf.Repositories[channelName]
		if !ok {
			return
		}

		c.mutex.Lock()
		channel, ok := c.channelMap[channelName]
		if !ok {
			channel = NewChannel(channelName, c.statikFS)
			waitGroup.Go(channel.Loop(ctx))
			c.channelMap[channelName] = channel
		}
		c.mutex.Unlock()

		channel.Push(ctx, &Event{hook: hook, repo: repo})
	}
}

// Channel 是用于处理 Webhook 事件的通道。
type Channel struct {
	name     string          // 仓库名
	statikFS http.FileSystem // 存储静态文件系统，用于加载静态资源
	cmd      *exec.Cmd       //  执行 air 命令的 *exec.Cmd 对象
	c        chan *Event     // Webhook 事件通道
}

// Event 代表一个 Webhook 事件
type Event struct {
	hook *webhook.Webhook
	repo *config.RepositoryConfig
}

func NewChannel(name string, statikFS http.FileSystem) *Channel {
	channel := &Channel{name: name, statikFS: statikFS, c: make(chan *Event, 16)}
	return channel
}

// Push 将 Webhook 事件推送到通道中进行处理。
func (c *Channel) Push(ctx context.Context, event *Event) {
	select {
	case <-ctx.Done():
		return
	case c.c <- event:
	}
}

// Loop 是一个循环函数，用于处理通道中的 Webhook 事件。
func (c *Channel) Loop(ctx context.Context) func() error {
	return func() error {
		for {
			select {
			case <-ctx.Done():
				_ = c.cmd.Process.Kill()
				return ctx.Err()
			case e := <-c.c:
				c.hotDeploy(e)
			}
		}
	}
}

// hotDeploy 是热部署处理的逻辑，包括克隆仓库、更新仓库、提取 air 二进制文件、启动 air 命令等操作。
func (c *Channel) hotDeploy(event *Event) {
	hook := event.hook
	repoDir := hook.Repository.Name

	// 检查仓库目录是否存在，如果不存在则执行 git clone 命令进行克隆
	if !fsutil.PathExists(repoDir) {
		err := shell.Run(fmt.Sprintf("git clone %s", hook.Repository.URL))
		if err != nil {
			log.Error().Err(err).Msgf("failed to git clone %s", hook.Repository.URL)
			return
		}
	}

	// 执行 git pull 命令更新仓库
	err := shell.Run(fmt.Sprintf("git pull %s", hook.Repository.URL), shell.WithDir(repoDir))
	if err != nil {
		log.Error().Err(err).Msgf("failed to git pull %s", hook.Repository.URL)
		return
	}

	// 检查 air 二进制文件是否存在，如果不存在则从静态文件系统中提取
	if !fsutil.FileExists(filepath.Join(repoDir, "air")) {
		err = c.extractAir(repoDir)
		if err != nil {
			log.Error().Err(err).Msg("failed to extract air binary")
			return
		}
	}

	// 如果进程未启动或已经退出，则启动 air 命令
	if c.cmd == nil || (c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited()) {
		cmd := shell.Command(fmt.Sprintf(`./air -build.cmd="%s" -build.args_bin="%s"`,
			event.repo.BuildCmd(), event.repo.BuildArgsBin), shell.WithDir(repoDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err = cmd.Start(); err != nil {
			log.Error().Err(err).Msg("failed to start air command")
			return
		}

		c.cmd = cmd

		// 等待 air 命令退出
		if err = cmd.Wait(); err != nil {
			log.Warn().Err(err).Msgf("failed to wait air command exit")
		}
	}
}

// extractAir 从静态文件系统中提取 air 二进制文件到目标目录（dstDir）。
func (c *Channel) extractAir(dstDir string) error {
	const airTarFilename = "air.tar.gz"

	// 打开静态文件系统中的 air.tar.gz 文件
	srcFile, err := c.statikFS.Open("/" + airTarFilename)
	if err != nil {
		return fmt.Errorf("statikFS Open failed: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(airTarFilename)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %w", airTarFilename, err)
	}
	defer func() {
		_ = dstFile.Close()
		_ = os.Remove(airTarFilename)
	}()

	// 将静态文件系统中的 air.tar.gz 文件复制到目标文件
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy '%s': %w", airTarFilename, err)
	}

	// 解压缩 air.tar.gz 文件到目标目录
	err = tarutil.Extract(airTarFilename, dstDir)
	if err != nil {
		return fmt.Errorf("failed to extract air bin: %w", err)
	}

	return nil
}
