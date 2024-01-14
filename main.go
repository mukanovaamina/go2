package main

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

type Task struct {
	gorm.Model
	Title     string
	Completed bool
}

var db *gorm.DB

func init() {
	dsn := "user=postgres password=Aruzhan7 dbname=amina sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto Migrate - Создание таблицы "tasks" в базе данных
	db.AutoMigrate(&Task{})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add-task", addTaskHandler)
	http.HandleFunc("/get-task", getTaskHandler)
	http.HandleFunc("/update-task", updateTaskHandler)
	http.HandleFunc("/delete-task", deleteTaskHandler)

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := getAllTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Task List</title>
			<style>
				.task {
					color: green;
					margin-bottom: 10px;
				}
			</style>
		</head>
		<body>
			<h1>Task List</h1>
			<ul>
				{{range .}}
					<li class="task">{{.Title}}</li>
				{{end}}
			</ul>
			<form action="/add-task" method="post">
				<label for="title">New Task:</label>
				<input type="text" id="title" name="title" required>
				<button type="submit">Add Task</button>
			</form>
			<form action="/get-task" method="post">
				<label for="taskId">Get Task by ID:</label>
				<input type="number" id="taskId" name="taskId" required>
				<button type="submit">Get Task</button>
			</form>
			<form action="/update-task" method="post">
				<label for="updateTaskId">Update Task by ID:</label>
				<input type="number" id="updateTaskId" name="updateTaskId" required>
				<label for="newTitle">New Title:</label>
				<input type="text" id="newTitle" name="newTitle" required>
				<button type="submit">Update Task</button>
			</form>
			<form action="/delete-task" method="post">
				<label for="deleteTaskId">Delete Task by ID:</label>
				<input type="number" id="deleteTaskId" name="deleteTaskId" required>
				<button type="submit">Delete Task</button>
			</form>
		</body>
		</html>
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, tasks)
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		err := addTask(title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		taskID := r.FormValue("taskId")
		id, err := strconv.Atoi(taskID)
		if err != nil {
			http.Error(w, "Invalid Task ID", http.StatusBadRequest)
			return
		}

		task, err := getTaskByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Task ID: %d, Title: %s, Completed: %t", task.ID, task.Title, task.Completed)
	}
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		taskID := r.FormValue("updateTaskId")
		newTitle := r.FormValue("newTitle")

		id, err := strconv.Atoi(taskID)
		if err != nil {
			http.Error(w, "Invalid Task ID", http.StatusBadRequest)
			return
		}

		err = updateTaskByID(id, newTitle)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		taskID := r.FormValue("deleteTaskId")
		id, err := strconv.Atoi(taskID)
		if err != nil {
			http.Error(w, "Invalid Task ID", http.StatusBadRequest)
			return
		}

		err = deleteTaskByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// CRUD операции для работы с задачами

func addTask(title string) error {
	task := Task{Title: title}
	result := db.Create(&task)
	return result.Error
}

func getTaskByID(taskID int) (Task, error) {
	var task Task
	result := db.First(&task, taskID)
	return task, result.Error
}

func updateTaskByID(taskID int, newTitle string) error {
	var task Task
	result := db.First(&task, taskID)
	if result.Error != nil {
		return result.Error
	}

	task.Title = newTitle
	result = db.Save(&task)
	return result.Error
}

func deleteTaskByID(taskID int) error {
	var task Task
	result := db.Delete(&task, taskID)
	return result.Error
}

func getAllTasks() ([]Task, error) {
	var tasks []Task
	result := db.Find(&tasks)
	return tasks, result.Error
}
