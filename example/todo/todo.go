package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

type todo struct {
	Num       int
	Content   string
	Done      bool
	CreatedAt time.Time
}

func (t todo) String() string {
	done := "[ ]"
	if t.Done {
		done = "[*]"
	}
	return fmt.Sprintf("%d: %s %s", t.Num, done, t.Content)
}

type todoList []todo

func (list *todoList) Load(fileName string) error {
	if list == nil {
		panic("list is nil")
	}

	*list = (*list)[:0]

	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(content, list)
	if err != nil {
		return err
	}

	sort.Slice(*list, func(i, j int) bool {
		// order: yet to done -> done
		if (*list)[i].Done != (*list)[j].Done {
			return !(*list)[i].Done
		}
		return (*list)[i].Num < (*list)[j].Num
	})

	return nil
}

func (list *todoList) Save(fileName string) error {
	if list == nil {
		panic("list is nil")
	}

	// re-number
	sort.Slice(*list, func(i, j int) bool {
		// order: yet to done -> done
		if (*list)[i].Done != (*list)[j].Done {
			return !(*list)[i].Done
		}
		return (*list)[i].CreatedAt.Before((*list)[j].CreatedAt)
	})
	for i := range *list {
		(*list)[i].Num = i + 1
	}

	content, err := json.MarshalIndent(*list, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileName, content, os.ModePerm)
}
