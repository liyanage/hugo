// Copyright © 2013 Steve Francia <spf@spf13.com>.
//
// Licensed under the Simple Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://opensource.org/licenses/Simple-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/spf13/cobra"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var serverPort int
var serverWatch bool

func init() {
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 1313, "port to run the server on")
	serverCmd.Flags().BoolVarP(&serverWatch, "watch", "w", false, "watch filesystem for changes and recreate as needed")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Hugo runs it's own a webserver to render the files",
	Long: `Hugo is able to run it's own high performance web server.
Hugo will render all the files defined in the source directory and
Serve them up.`,
	Run: server,
}

func server(cmd *cobra.Command, args []string) {
	InitializeConfig()

	if Config.BaseUrl == "" {
		Config.BaseUrl = "http://localhost:" + strconv.Itoa(serverPort)
	}

	// Watch runs its own server as part of the routine
	if serverWatch {
		fmt.Println("Watching for changes in", Config.GetAbsPath(Config.ContentDir))
		err := NewWatcher(serverPort, true)
		if err != nil {
			fmt.Println(err)
		}
	}

	serve(serverPort)
}

func serve(port int) {
	if Verbose {
		fmt.Println("Serving pages from " + Config.GetAbsPath(Config.PublishDir))
	}

	fmt.Println("Web Server is available at http://localhost:", port)
	fmt.Println("Press ctrl+c to stop")
	panic(http.ListenAndServe(":"+strconv.Itoa(port), http.FileServer(http.Dir(Config.GetAbsPath(Config.PublishDir)))))
}

func NewWatcher(port int, server bool) error {
	watcher, err := fsnotify.NewWatcher()
	var wg sync.WaitGroup

	if err != nil {
		fmt.Println(err)
		return err
	}

	defer watcher.Close()

	wg.Add(1)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if Verbose {
					fmt.Println(ev)
				}
				watchChange(ev)
				// TODO add newly created directories to the watch list
			case err := <-watcher.Error:
				if err != nil {
					fmt.Println("error:", err)
				}
			}
		}
	}()

	for _, d := range getDirList() {
		if d != "" {
			_ = watcher.Watch(d)
		}
	}

	if server {
		go serve(port)
	}

	wg.Wait()
	return nil
}

func watchChange(ev *fsnotify.FileEvent) {
	if strings.HasPrefix(ev.Name, Config.GetAbsPath(Config.StaticDir)) {
		fmt.Println("Static file changed, syncing\n")
		copyStatic()
	} else {
		fmt.Println("Change detected, rebuilding site\n")
		buildSite()
	}
}