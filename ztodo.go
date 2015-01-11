package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/jack-zh/ztodo/task"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var noAct = errors.New("error")

var version = "ztodo version 0.4.4 (2015-01-11 build)"

var userconfig_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "userconfig.json")
var cloud_work_tasks_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "worktasks.json")
var cloud_backup_tasks_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "backuptasks.json")
var simple_tasks_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "simpletasks")

const usage = `Usage:
	ztodo version
		Show ztodo version
	ztodo list|ls
		Show all tasks
	ztodo list|ls N
		Show task N
	ztodo rm|remove N
		Remove task N
	ztodo done N
		Done task N
	ztodo undo N
		Undo task N
	ztodo doing N
		Doing task N
	ztodo clean
		Rm done task
	ztodo clear
		Rm all task
	ztodo add ...
		Add task to list
`

func printSimpleTask(t string, i string) {
	if strings.HasPrefix(t, "0") {
		t = strings.Replace(t, "0", "[Future]", 1)
	}
	if strings.HasPrefix(t, "1") {
		t = strings.Replace(t, "1", "[Doing ]", 1)
	}
	if strings.HasPrefix(t, "2") {
		t = strings.Replace(t, "2", "[Done  ]", 1)
	}
	fmt.Printf("%2s: %s\n", i, t)
}

func dirCheck() error {
	var filename = filepath.Join(os.Getenv("HOME"), ".ztodo")
	finfo, err := os.Stat(filename)
	if err != nil {
		os.Mkdir(filename, os.ModePerm)
		return nil
	}
	if finfo.IsDir() {
		return nil
	} else {
		return errors.New("$HOME/.ztodo is a file not dir.")
	}
}

func main() {
	errdir := dirCheck()
	if errdir != nil {
		os.Exit(1)
	}
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	simplelist := task.SimpleNewList(simple_tasks_filename)
	cloudlist := task.CloudNewList(cloud_work_tasks_filename, cloud_backup_tasks_filename, userconfig_filename)
	a, n := flag.Arg(0), len(flag.Args())

	a = strings.ToLower(a)
	if a == "ls" {
		a = "list"
	}
	if a == "remove" {
		a = "rm"
	}

	err := noAct
	switch {
	case a == "version" && n == 1:
		fmt.Println(version)
		err = nil

	case a == "help" && n == 1:
		fmt.Println(usage)
		err = nil

	case a == "simplelist" && n == 1:
		var tasks []string
		tasks, err = simplelist.SimpleGet()
		for i := 0; i < len(tasks); i++ {
			printSimpleTask(tasks[i], strconv.Itoa(i+1))
		}
	case a == "simplelist" && n == 2:
		i, err2 := strconv.Atoi(flag.Arg(1))
		if err2 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		var task string
		task, err = simplelist.SimpleGetTask(i - 1)
		if err == nil {
			printSimpleTask(task, strconv.Itoa(i))
		}
	case a == "simplerm" && n == 2:
		i, err2 := strconv.Atoi(flag.Arg(1))
		if err2 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = simplelist.SimpleRemoveTask(i - 1)
		if err != nil {
			break
		}
	case a == "simpleadd" && n > 1:
		t := strings.Join(flag.Args()[1:], " ")
		err = simplelist.SimpleAddTask(t)
		err = cloudlist.CloudAddTask(t)

	case a == "simpledoing" && n == 2:
		i, err3 := strconv.Atoi(flag.Args()[1])
		if err3 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = simplelist.SimpleDoingTask(i - 1)

	case a == "simpledone" && n == 2:
		i, err4 := strconv.Atoi(flag.Args()[1])
		if err4 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = simplelist.SimpleDoneTask(i - 1)
	case a == "simpleundo" && n == 2:
		i, err5 := strconv.Atoi(flag.Args()[1])
		if err5 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = simplelist.SimpleUndoTask(i - 1)
	case a == "simpleclean" && n == 1:
		err = simplelist.SimpleCleanTask()
	case a == "simpleclear" && n == 1:
		err = simplelist.SimpleClearTask()

	case a == "list" && n == 1:
		err = cloudlist.CloudGetAllWorkTaskByFile()
		if err == nil {
			cloudlist.CloudTasksPrint(-1)
		}

	case a == "list" && n == 2:
		if flag.Arg(1) == "verbose" {
			err = cloudlist.CloudGetAllWorkTaskByFile()
			if err == nil {
				cloudlist.CloudTasksPrintVerbose(-1)
			}
		} else {
			i, err2 := strconv.Atoi(flag.Arg(1))
			if err2 != nil {
				fmt.Fprint(os.Stdout, usage)
				break
			}
			err = cloudlist.CloudGetAllWorkTaskByFile()
			if err == nil {
				cloudlist.CloudTasksPrint(i)
			}
		}
	case a == "list" && n == 3:
		if flag.Arg(2) == "verbose" {
			i, err2 := strconv.Atoi(flag.Arg(1))
			if err2 != nil {
				fmt.Fprint(os.Stdout, usage)
				break
			}
			err = cloudlist.CloudGetAllWorkTaskByFile()
			if err == nil {
				cloudlist.CloudTasksPrintVerbose(i)
			}
		} else {
			fmt.Fprint(os.Stdout, usage)
		}

	case a == "rm" && n == 2:
		i, err2 := strconv.Atoi(flag.Arg(1))
		if err2 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = cloudlist.CloudRemoveTask(i)
		if err != nil {
			break
		}
	case a == "add" && n > 1:
		t := strings.Join(flag.Args()[1:], " ")
		err = cloudlist.CloudAddTask(t)

	case a == "doing" && n == 2:
		i, err3 := strconv.Atoi(flag.Args()[1])
		if err3 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = cloudlist.CloudDoingTask(i)

	case a == "done" && n == 2:
		i, err4 := strconv.Atoi(flag.Args()[1])
		if err4 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = cloudlist.CloudDoneTask(i)
	case a == "undo" && n == 2:
		i, err5 := strconv.Atoi(flag.Args()[1])
		if err5 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = cloudlist.CloudUndoTask(i)
	case a == "clean" && n == 1:
		err = cloudlist.CloudCleanTask()
	case a == "clear" && n == 1:
		err = cloudlist.CloudClearTask()
	default:
		fmt.Fprint(os.Stdout, usage)
	}
	if err != nil {
		fmt.Println(err)
	} else {
		if a != "list" && a != "version" && a != "help" {
			fmt.Println("\nSuccess!\n")
		}
	}
}
