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

var version = "ztodo version 0.4 (2015-01-05 build)"

var userconfig_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "userconfig.json")
var tasks_filename = filepath.Join(os.Getenv("HOME"), ".ztodo", "tasks.json")

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

func printTask(task string, i string) {
	if strings.HasPrefix(task, "0") {
		task = strings.Replace(task, "0", "[Future]", 1)
	}
	if strings.HasPrefix(task, "1") {
		task = strings.Replace(task, "1", "[Doing ]", 1)
	}
	if strings.HasPrefix(task, "2") {
		task = strings.Replace(task, "2", "[Done  ]", 1)
	}
	fmt.Printf("%2s: %s\n", i, task)
}

func dirCheck() error {
	var filename = filepath.Join(os.Getenv("HOME"), ".ztodo")
	finfo, err := os.Stat(filename)
	if err != nil {
		os.Mkdir(filename, 0)
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

	list := task.NewList(tasks_filename)
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

	case a == "list" && n == 1:
		var tasks []string
		tasks, err = list.Get()
		for i := 0; i < len(tasks); i++ {
			printTask(tasks[i], strconv.Itoa(i+1))
		}

	case a == "list" && n == 2:
		i, err2 := strconv.Atoi(flag.Arg(1))
		if err2 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		var task string
		task, err = list.GetTask(i - 1)
		if err == nil {
			printTask(task, strconv.Itoa(i))
		}
	case a == "rm" && n == 2:
		i, err2 := strconv.Atoi(flag.Arg(1))
		if err2 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = list.RemoveTask(i - 1)
		if err != nil {
			break
		}
	case a == "add" && n > 1:
		t := strings.Join(flag.Args()[1:], " ")
		err = list.AddTask(t)

	case a == "doing" && n == 2:
		i, err3 := strconv.Atoi(flag.Args()[1])
		if err3 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = list.DoingTask(i - 1)

	case a == "done" && n == 2:
		i, err4 := strconv.Atoi(flag.Args()[1])
		if err4 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = list.DoneTask(i - 1)
	case a == "undo" && n == 2:
		i, err5 := strconv.Atoi(flag.Args()[1])
		if err5 != nil {
			fmt.Fprint(os.Stdout, usage)
			break
		}
		err = list.UndoTask(i - 1)
	case a == "clean" && n == 1:
		err = list.CleanTask()
	case a == "clear" && n == 1:
		err = list.ClearTask()
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
