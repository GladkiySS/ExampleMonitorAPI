package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func main() {
	// Перевіряємо командний рядок на наявність аргументів
	if len(os.Args) > 1 {
		// Обробка командних аргументів
		switch command := strings.ToLower(os.Args[1]); {
		case command == "--help":
			printHelp()

		case command == "--createdb":
			// Створюємо базу даних та додаємо дані з файлу
			_, err := os.Stat("./monitors.txt")

			if os.IsNotExist(err) {
				fmt.Println("ERROR! File \"monitors.txt\" does not exist!")
				return
			}

			_, err = os.Stat("./products.db")

			if err == nil {
				err = os.Remove("./products.db")

				if err != nil {
					fmt.Println(err)
					return
				}
			}

			CreateDB()

			AddMonitorsFromFile("./monitors.txt")

			fmt.Println("OK. File products.db is created!")

			return

		case command == "--start":
			// Запускаємо веб-сервер
			http.HandleFunc("/category/monitors", GetMonitors)
			http.HandleFunc("/category/monitor/", GetStatForMonitor)
			http.HandleFunc("/category/monitor_click/", AddClickForMonitor)
			http.HandleFunc("/", MainPage)

			fmt.Println("The server is running!")
			fmt.Println("Looking forward to request...")

			err := http.ListenAndServe(":8030", nil)

			if err != nil {
				log.Fatal("Failed to start server!", err)
			}

		default:
			printHelp()
		}
	} else {
		printHelp()
	}
}

func CreateDB() {
	// Відкриваємо з'єднання з базою даних
	OpenDB()

	// Створюємо таблицю "monitors" з трьома колонками
	_, err := DB.Exec("create table monitors(id integer, name varchar(255) not null, count integer)")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}

	// Закриваємо з'єднання з базою даних
	DB.Close()
}

func OpenDB() {
	// Відкриваємо з'єднання з базою даних SQLite
	db, err := sql.Open("sqlite3", "products.db")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	DB = db
}

func printHelp() {
	// Виводимо довідку про доступні команди
	fmt.Println()
	fmt.Println("Help:	./counter --help")
	fmt.Println("Create products database:  ./counter --createdb")
	fmt.Println("Start server:	./counter --start")
	fmt.Println()
}

func AddMonitorsFromFile(filename string) {
	// Відкриваємо файл для читання
	var file *os.File
	var err error

	file, err = os.Open(filename)

	if err != nil {
		log.Fatal("Failed to open the file: ", err)
		os.Exit(2)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	OpenDB()

	// Зчитуємо дані з файлу та додаємо їх до таблиці "monitors"
	for scanner.Scan() {
		arr := strings.Split(scanner.Text(), ",")
		id := arr[0]
		monitorName := arr[1]
		_, err = DB.Exec("insert into monitors(id, name, count) values($1, $2, 0)", id, monitorName)
	}
}

func AddClickForMonitor(w http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()

	if err != nil {
		fmt.Fprintf(w, "{%s}", err)
	} else {
		// Отримуємо ідентифікатор монітора та збільшуємо лічильник кліків
		monitorId := strings.TrimPrefix(request.URL.Path, "/category/monitor_click/")
		OpenDB()
		countValue := 0
		rows, _ := DB.Query("select count from monitors where id=" + monitorId)

		for rows.Next() {
			rows.Scan(&countValue)
		}

		countValue++
		_, err = DB.Exec("update monitors set count=" + strconv.Itoa(countValue) + " where id=" + monitorId)
	}
}

func GetFromDBNameModel(tblName string) []string {
	var arr []string
	var monitorName string

	// Отримуємо імена моніторів з бази даних
	rows, _ := DB.Query("select name from " + tblName)

	for rows.Next() {
		rows.Scan(&monitorName)
		arr = append(arr, monitorName)
	}

	return arr
}

func GetMonitors(w http.ResponseWriter, request *http.Request) {
	OpenDB()

	// Отримуємо список імен моніторів та відправляємо їх у форматі JSON
	monitors := GetFromDBNameModel("monitors")
	err := request.ParseForm()

	if err != nil {
		fmt.Fprintf(w, "{%s}", err)
	} else {
		var monitorList []interface{}

		for i := range monitors {
			monitorList = append(monitorList, []interface{}{i + 1, monitors[i]})
		}

		response := map[string]interface{}{
			"monitors": monitorList,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			fmt.Fprintf(w, "{%s}", err)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
		}
	}
}

func GetStatForMonitor(w http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()

	if err != nil {
		fmt.Fprintf(w, "{%s}", err)
	} else {
		// Отримуємо лічильник кліків для певного монітора та відправляємо його відповідь
		countValue := 0
		monitorId := strings.TrimPrefix(request.URL.Path, "/category/monitor/")

		OpenDB()

		rows, _ := DB.Query("select count from monitors where id=" + monitorId)

		for rows.Next() {
			rows.Scan(&countValue)
		}

		strOut := "{ \"id\": \"" + monitorId + "\", \"count\": \"" + strconv.Itoa(countValue) + "\"}"

		fmt.Fprintf(w, strOut)
	}
}

func MainPage(w http.ResponseWriter, r *http.Request) {
	// Обробка головної сторінки
	http.ServeFile(w, r, "TestDB.html")
}
