package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/porter-dev/porter/cli/cmd/docker"
	"k8s.io/client-go/util/homedir"

	"github.com/porter-dev/porter/cli/cmd/credstore"

	"github.com/spf13/cobra"

	"github.com/docker/docker/api/types/mount"
)

type startOps struct {
	insecure       *bool
	skipKubeconfig *bool
	kubeconfigPath string
	contexts       *[]string
	imageTag       string `form:"required"`
	db             string `form:"oneof=sqlite postgres"`
}

var opts = &startOps{}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Use:   "start",
	Short: "Starts a Porter instance using the Docker engine.",
	Run: func(cmd *cobra.Command, args []string) {
		closeHandler(stop)

		err := start(
			opts.imageTag,
			opts.kubeconfigPath,
			opts.db,
			*opts.contexts,
			*opts.insecure,
			*opts.skipKubeconfig,
		)

		if err != nil {
			fmt.Println("Error running start:", err.Error())
			fmt.Println("Shutting down...")

			err = stop()

			if err != nil {
				fmt.Println("Shutdown unsuccessful:", err.Error())
			}

			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	opts.insecure = startCmd.PersistentFlags().Bool(
		"insecure",
		false,
		"skip admin setup and authorization",
	)

	opts.skipKubeconfig = startCmd.PersistentFlags().Bool(
		"skip-kubeconfig",
		false,
		"skip initialization of the kubeconfig",
	)

	opts.contexts = startCmd.PersistentFlags().StringArray(
		"contexts",
		nil,
		"the list of contexts to use (defaults to the current context)",
	)

	startCmd.PersistentFlags().StringVar(
		&opts.db,
		"db",
		"sqlite",
		"the db to use, one of sqlite or postgres",
	)

	startCmd.PersistentFlags().StringVar(
		&opts.kubeconfigPath,
		"kubeconfig",
		"",
		"path to kubeconfig",
	)

	startCmd.PersistentFlags().StringVar(
		&opts.imageTag,
		"image-tag",
		"latest",
		"the Porter image tag to use",
	)
}

func stop() error {
	agent, err := docker.NewAgentFromEnv()

	if err != nil {
		return err
	}

	err = agent.StopPorterContainers()

	if err != nil {
		return err
	}

	return nil
}

func start(
	imageTag string,
	kubeconfigPath string,
	db string,
	contexts []string,
	insecure bool,
	skipKubeconfig bool,
) error {
	var username, pw string
	var err error
	home := homedir.HomeDir()
	outputConfPath := filepath.Join(home, ".porter", "porter.kubeconfig")
	containerConfPath := "/porter/porter.kubeconfig"
	port := 8080

	// if not insecure, or username/pw set incorrectly, prompt for new username/pw
	if username, pw, err = credstore.Get(); !insecure && err != nil {
		fmt.Println("Please register your admin account with an email and password:")

		username, err = promptPlaintext("Email: ")

		if err != nil {
			return err
		}

		pw, err = promptPasswordWithConfirmation()

		if err != nil {
			return err
		}

		credstore.Set(username, pw)
	}

	if !skipKubeconfig {
		err = generate(
			kubeconfigPath,
			outputConfPath,
			false,
			contexts,
		)

		if err != nil {
			return err
		}
	}

	agent, err := docker.NewAgentFromEnv()

	if err != nil {
		return err
	}

	// the volume mounts to use
	mounts := make([]mount.Mount, 0)

	// the volumes passed to the Porter container
	volumesMap := make(map[string]struct{})

	if !skipKubeconfig {
		// add a bind mount with the kubeconfig
		mount := mount.Mount{
			Type:        mount.TypeBind,
			Source:      outputConfPath,
			Target:      containerConfPath,
			ReadOnly:    true,
			Consistency: mount.ConsistencyFull,
		}

		mounts = append(mounts, mount)
	}

	netID, err := agent.CreateBridgeNetworkIfNotExist("porter_network")

	if err != nil {
		return err
	}

	env := make([]string, 0)

	env = append(env, []string{
		"ADMIN_INIT=true",
		"ADMIN_EMAIL=" + username,
		"ADMIN_PASSWORD=" + pw,
	}...)

	switch db {
	case "sqlite":
		// check if sqlite volume exists, create it if not
		vol, err := agent.CreateLocalVolumeIfNotExist("porter_sqlite")

		if err != nil {
			return err
		}

		// create mount
		mount := mount.Mount{
			Type:        mount.TypeVolume,
			Source:      vol.Name,
			Target:      "/sqlite",
			ReadOnly:    false,
			Consistency: mount.ConsistencyFull,
		}

		mounts = append(mounts, mount)
		volumesMap[vol.Name] = struct{}{}

		env = append(env, []string{
			"SQL_LITE=true",
			"SQL_LITE_PATH=/sqlite/porter.db",
		}...)
	case "postgres":
		// check if postgres volume exists, create it if not
		vol, err := agent.CreateLocalVolumeIfNotExist("porter_postgres")

		if err != nil {
			return err
		}

		// pgMount is mount for postgres container
		pgMount := []mount.Mount{
			mount.Mount{
				Type:        mount.TypeVolume,
				Source:      vol.Name,
				Target:      "/var/lib/postgresql/data",
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
			},
		}

		// create postgres container with mount
		startOpts := docker.PostgresOpts{
			Name:   "porter_postgres",
			Image:  "postgres:latest",
			Mounts: pgMount,
			VolumeMap: map[string]struct{}{
				"porter_postgres": struct{}{},
			},
			NetworkID: netID,
			Env: []string{
				"POSTGRES_USER=porter",
				"POSTGRES_PASSWORD=porter",
				"POSTGRES_DB=porter",
			},
		}

		pgID, err := agent.StartPostgresContainer(startOpts)

		fmt.Println("Waiting for postgres:latest to be healthy...")
		agent.WaitForContainerHealthy(pgID, 10)

		if err != nil {
			return err
		}

		env = append(env, []string{
			"SQL_LITE=false",
			"DB_USER=porter",
			"DB_PASS=porter",
			"DB_NAME=porter",
			"DB_HOST=porter_postgres",
			"DB_PORT=5432",
		}...)

		defer agent.WaitForContainerStop(pgID)
	}

	// create Porter container
	// TODO -- look for unused port
	startOpts := docker.PorterStartOpts{
		Name:          "porter_server",
		Image:         "porter1/porter:" + imageTag,
		HostPort:      uint(port),
		ContainerPort: 8080,
		Mounts:        mounts,
		VolumeMap:     volumesMap,
		NetworkID:     netID,
		Env:           env,
	}

	id, err := agent.StartPorterContainer(startOpts)

	if err != nil {
		return err
	}

	fmt.Println("Spinning up the server...")
	time.Sleep(7 * time.Second)
	openBrowser(fmt.Sprintf("http://localhost:%d/login?email=%s", port, username))
	fmt.Printf("Server ready: listening on localhost:%d\n", port)

	agent.WaitForContainerStop(id)

	return nil
}

// openBrowser opens the specified URL in the default browser of the user.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
