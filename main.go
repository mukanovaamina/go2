package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

// Task структура представляет задачу
type Task struct {
	ID        int
	Title     string
	Completed bool
}

var db *sql.DB

func init() {
	// Замените параметры подключения вашей базы данных PostgreSQL
	connStr := "user=postgres password=Aruzhan7 dbname=amina sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add-task", addTaskHandler)

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем список задач из базы данных
	tasks, err := getTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Загружаем HTML-шаблон и передаем список задач
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
		</body>
		</html>
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем HTML-страницу с задачами в ответе
	tmpl.Execute(w, tasks)
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Обработка добавления задачи
	if r.Method == http.MethodPost {
		title := r.FormValue("title")

		// Добавляем задачу в базу данных
		err := addTask(title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Перенаправляем пользователя обратно на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getTasks() ([]Task, error) {
	rows, err := db.Query("SELECT id, title, completed FROM tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Title, &task.Completed)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func addTask(title string) error {
	_, err := db.Exec("INSERT INTO tasks (title, completed) VALUES ($1, false)", title)
	return err
}
