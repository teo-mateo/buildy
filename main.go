package main

import (
	"fmt"
	"flag"
	"os"
	"github.com/fsnotify/fsnotify"
	"log"
	"time"
	"bufio"
	"strings"
	"errors"
	"os/exec"
)

func readLastLine(f string) (line string, err error) {
	file, err := os.Open(f)
	if err != nil{
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		line = scanner.Text()
	}

	if err := scanner.Err(); err != nil{
		fmt.Println(err)
		os.Exit(1)
	}

	return line, nil
}

func parseLine(l string) (ts time.Time, container string, err error){
	splits := strings.Split(l, "|")

	if len(splits) != 2{
		return time.Time{}, "", errors.New("bad format")
	}

	ts, err = time.Parse("20060102030405", splits[0])
	if err != nil{
		return time.Time{}, "", err
	}

	if len(strings.Trim(splits[1], " ")) == 0 {
		return time.Time{}, "", errors.New("bad format")
	}

	container = splits[1]

	return

}

func main(){
	fmt.Println("Hello fish")

	var f string
	var cmdString string

	flag.StringVar(&f, "file", "", "--file /file/to/watch")
	flag.StringVar(&cmdString, "cmd", "~/containers/{container}/build.sh", "--cmd=<some-command{container}>")
	flag.Parse()

	if f == ""{
		fmt.Println("missing file argument.")
		os.Exit(1)
	}

	if cmdString == ""{
		fmt.Println("missing cmd argument")
		os.Exit(1)
	}

	if _, err := os.Stat(f); os.IsNotExist(err){
		fmt.Println("file does not exist.")
		os.Exit(1)
	}

	fmt.Printf("File: %s\n", f)
	fmt.Printf("Cmd: %s\n", cmdString)

	watcher, err := fsnotify.NewWatcher()
	if err != nil{
		fmt.Println(err)
		os.Exit(1)
	}
	defer watcher.Close()

	lastChange := time.Now()

	done := make(chan bool)
	go func(){
		for {
			select{
			case event := <- watcher.Events:
				if time.Since(lastChange).Seconds() > 1{
					log.Println("event:",event)
					if event.Op&fsnotify.Write == fsnotify.Write{
						log.Println("file was modified")

						lastLine, err := readLastLine(f)
						if err != nil{
							log.Println(err)
							os.Exit(1)
						}
						fmt.Printf("Last line: %s\n", lastLine)

						ts, container, err := parseLine(lastLine)
						if err != nil{
							log.Println(err)
							os.Exit(1)
						}
						fmt.Printf("%v %s\n", ts, container)

						cmdString = strings.Replace(cmdString, "{container}", container, 1)
						cmd := exec.Command(cmdString)
						err = cmd.Run()
						if err != nil{
							fmt.Println(err)
						}
					}
					lastChange = time.Now()
				}
			case err := <- watcher.Errors:
				log.Println("error:", err)
				os.Exit(1)
			}
		}
	}()


	err = watcher.Add(f)
	if err != nil{
		fmt.Println(err)
		os.Exit(1)
	}

	<-done
}