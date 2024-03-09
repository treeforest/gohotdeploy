# GoHotDeploy

GoHotDeploy 是一个轻量级工具，通过 GitLab Webhooks 实现 Go 应用程序的热部署。

[English](https://github.com/treeforest/gohotdeploy/blob/main/README.md) | 简体中文

## 使用方法

1. 安装 GoHotDeploy：

   ```shell
   go get github.com/treeforest/gohotdeploy
   ```

2. 创建一个名为 `config.yml` 的配置文件，内容如下：

   ```yaml
   port: 8080
   repositories:
     - name: my-repo
       relative_build_dir: .
       run_args: ""
   ```

   将 `.` 替换为 Git 仓库中构建目录的相对路径。如果 `relative_build_dir` 为空，则默认为当前仓库目录。

   修改 `run_args` 的值以包含在执行构建的二进制文件时传递的参数。如果 `run_args` 为空，则在程序执行时不会传递额外的参数。

3. 启动 GoHotDeploy：

   ```shell
   gohotdeploy --config=config.yml
   ```

   GoHotDeploy 将在指定的端口（默认为 8080）上启动一个 HTTP 服务器，用于监听 GitLab Webhook 事件。

4. 配置 GitLab Webhook：

   - 进入 GitLab 项目设置页面。
   - 在左侧导航栏中选择 "Webhooks"。
   - 添加一个新的 Webhook，设置如下：
     - URL: `http://your-server-ip:8080/`
     - 触发器: "Push events"
     - SSL 验证: 根据服务器配置启用或禁用
     - 保存 Webhook。

   每当 GitLab 仓库发生推送事件时，GoHotDeploy 将自动构建和部署您的应用程序。

## 许可证

[GNU General Public License v3.0](https://github.com/treeforest/gohotdeploy/blob/main/LICENSE)