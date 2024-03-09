# GoHotDeploy

GoHotDeploy is a lightweight tool that enables hot deployment of Go applications using GitLab Webhooks.

English | 简体中文

## Usage

1. Install GoHotDeploy:    

   ```shell
   go get github.com/treeforest/gohotdeploy
   ```

2. Create a configuration file `config.yml` with the following content:

   ```yaml
   port: 8080
   repositories:
     - name: my-repo
       relative_build_dir: .
       run_args: ""
   ```

   Replace `.` with the relative path to the build directory within your Git repository. If `relative_build_dir` is left empty, it defaults to the current directory of the repository.

   Modify the `run_args` value to include the desired arguments to be passed when executing the built binary. If `run_args` is left empty, no additional arguments will be passed during program execution.

3. Start GoHotDeploy:

   ```shell
   gohotdeploy --config=config.yml
   ```

   GoHotDeploy will start an HTTP server listening on the specified port (default: 8080) for GitLab webhook events.

4. Configure GitLab webhook:

   - Go to your GitLab project settings.
   - Navigate to "Webhooks" in the left sidebar.
   - Add a new webhook with the following settings:
     - URL: `http://your-server-ip:8080/`
     - Trigger: "Push events"
     - SSL verification: Enable or disable based on your server configuration
     - Save the webhook.

   Whenever a push event occurs in your GitLab repository, GoHotDeploy will automatically build and deploy your application.

## License

[GNU General Public License v3.0](https://github.com/cosmtrek/air/blob/v1.40.4/LICENSE)

