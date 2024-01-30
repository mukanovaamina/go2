package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var log = logrus.New()
var db *gorm.DB
var (
	limiter = rate.NewLimiter(1, 3) // Rate limit of 1 request per second with a burst of 3 requests
	mu      sync.Mutex
)

func init() {
	// Configure logrus
	log.SetFormatter(&logrus.JSONFormatter{})

	// Configure database
	dsn := "user=postgres password=Aruzhan7 dbname=amina sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto Migrate - Create "tasks" table in the database
	db.AutoMigrate(&Task{})
}

type Task struct {
	gorm.Model
	Title     string
	Completed bool
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(logrus.Fields{
		"action": "request",
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Received a request")

	// Rate Limiting
	if !limiter.Allow() {
		log.WithFields(logrus.Fields{
			"action": "error",
			"error":  "Rate limit exceeded",
		}).Error("Rate limit exceeded")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	filter := r.URL.Query().Get("filter")
	sort := r.URL.Query().Get("sort")
	page := r.URL.Query().Get("page")
	limit := 10 // Number of items per page
	offset := 0 // Offset for SQL query

	// Calculate offset based on pagination
	if p, err := strconv.Atoi(page); err == nil && p > 1 {
		offset = (p - 1) * limit
	}

	// SQL query considering all parameters
	query := "SELECT * FROM tasks"
	if filter != "" {
		query += " WHERE title LIKE '%" + filter + "%'"
	}
	if sort != "" {
		query += " ORDER BY " + sort
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	tasks, err := getTasksWithFilterAndSort(query)
	if err != nil {
		log.WithFields(logrus.Fields{
			"action": "error",
			"error":  err.Error(),
		}).Error("Error retrieving tasks")
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
				{{range .Tasks}}
					<li class="task">{{.Title}}</li>
				{{end}}
			</ul>
			<form action="/" method="get">
				<label for="filter">Filter by Title:</label>
				<input type="text" id="filter" name="filter" placeholder="Enter filter" value="{{.Filter}}">
				<button type="submit">Apply Filter</button>
			</form>
			<form action="/" method="get">
				<label for="sort">Sort by:</label>
				<select id="sort" name="sort">
					<option value="title">Title</option>
					<option value="created_at">Created At</option>
					<!-- Add more options as needed -->
				</select>
				<button type="submit">Apply Sorting</button>
			</form>
			<form action="/" method="get">
				<label for="page">Page:</label>
				<input type="number" id="page" name="page" min="1" value="{{.Page}}">
				<button type="submit">Go</button>
			</form>
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
		log.WithFields(logrus.Fields{
			"action": "error",
			"error":  err.Error(),
		}).Error("Error rendering template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, struct {
		Tasks  []Task
		Filter string
		Page   string
	}{
		Tasks:  tasks,
		Filter: filter,
		Page:   page,
	})

	log.WithFields(logrus.Fields{
		"action": "response",
		"status": http.StatusOK,
	}).Info("Request processed successfully")
}

func getTasksWithFilterAndSort(query string) ([]Task, error) {
	var tasks []Task
	result := db.Raw(query).Find(&tasks)
	return tasks, result.Error
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

func main() {
	http.HandleFunc("/", indexHandler)
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
