package business

import (
	"database/sql"
	"encoding/json"
	"github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type VideoInfo struct {
	ID        int
	EnName    string
	JpName    string
	Episode   string
	ImagePath string
	VideoPath string
}

type VideoServer struct {
	dataSource    string
	resourcesPath string
	database      *sql.DB
	server        *http.Server
}

func (vs *VideoServer) fetchResources(w http.ResponseWriter, r *http.Request) {
	rows, e := vs.database.Query("select * from tb_porn_index")
	if e != nil {
		log.Fatalln(e)
	}

	resources := make([]VideoInfo, 0)
	for rows.Next() {
		result := VideoInfo{}
		e := rows.Scan(
			&result.ID,
			&result.EnName,
			&result.JpName,
			&result.Episode,
			&result.ImagePath,
			&result.VideoPath,
		)
		if e != nil {
			log.Fatalln(e)
		}
		resources = append(resources, result)
	}

	response := make(map[string][]VideoInfo)
	for i := range resources {
		if _, exist := response[resources[i].EnName]; !exist {
			response[resources[i].EnName] = make([]VideoInfo, 0)
		}

		response[resources[i].EnName] = append(response[resources[i].EnName], resources[i])
	}

	bytes, e := json.Marshal(response)
	if e != nil {
		log.Fatalln(e)
	}

	w.Header().Set("Content-Type", "application/json")
	if i, e := w.Write(bytes); e != nil {
		log.Fatalln(e)
	} else {
		log.Println("Response Length ", i)
	}
}

func (vs *VideoServer) serveVideo(w http.ResponseWriter, r *http.Request) {
	id := -1
	if r.Method == "GET" {
		log.Println("Query for video = ", r.URL.Query()["id"])
		v, e := strconv.Atoi(r.URL.Query()["id"][0])
		if e != nil {
			log.Fatalln(e)
		}
		id = v
	}

	if id != -1 {
		videoPath := ""
		row := vs.database.QueryRow("select video_path from tb_porn_index where id = ?", id)
		if e := row.Scan(&videoPath); e != nil {
			log.Fatalln(e)
		}

		http.ServeFile(w, r, videoPath)
	}
}

func NewVideoServer(dataSource, resourcesPath string) (*VideoServer, error) {
	vs := new(VideoServer)
	vs.dataSource = dataSource
	vs.resourcesPath = resourcesPath

	config := mysql.NewConfig()
	config.Net = "tcp"
	config.Addr = "127.0.0.1:3306"
	config.User = "root"
	config.Passwd = "dev#pass"
	config.DBName = "db_data"

	driver, e := mysql.NewConnector(config)
	if e != nil {
		return nil, e
	}
	vs.database = sql.OpenDB(driver)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/fetchResources", vs.fetchResources)
	serverMux.HandleFunc("/serveVideo", vs.serveVideo)
	vs.server = &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}
	return vs, nil
}

func (vs *VideoServer) Close() {
	if e := vs.database.Close(); e != nil {
		panic(e)
	}

	if e := vs.server.Close(); e != nil {
		panic(e)
	}
}

func (vs *VideoServer) Serve() error {
	return vs.server.ListenAndServe()
}

func (vs *VideoServer) Index() error {
	files, e := ioutil.ReadDir(vs.resourcesPath)
	if e != nil {
		return e
	}

	videos := make([]VideoInfo, 0)
	for i := range files {
		if files[i].IsDir() {
			name := files[i].Name()
			jpName := ""
			resources := make(map[string]*VideoInfo)
			walker := func(path string, info os.FileInfo, err error) error {
				pattern := regexp.MustCompile(`^(.*)\.(mp4|jpg|png)$`)
				if info.IsDir() || !pattern.MatchString(info.Name()) {
					if info.Name() == "description.json" {
						description, e := ioutil.ReadFile(path)
						if e != nil {
							return e
						}
						videoDescription := make(map[string]string)
						if e := json.Unmarshal(description, &videoDescription); e != nil {
							return e
						}
						jpName = videoDescription["JpName"]
					}
					return nil
				}
				information := pattern.FindStringSubmatch(info.Name())
				resource, exist := resources[information[1]]
				if !exist {
					resource = &VideoInfo{
						EnName:  name,
						Episode: information[1],
					}
					resources[information[1]] = resource
				}
				if information[2] == "mp4" {
					resource.VideoPath = path
				} else {
					resource.ImagePath = path
				}
				return nil
			}
			if e = filepath.Walk(filepath.Join(vs.resourcesPath, name), walker); e != nil {
				return e
			}
			for k := range resources {
				resources[k].JpName = jpName
				videos = append(videos, *resources[k])
			}
		}
	}

	query := "insert into tb_porn_index values(NULL, ?, ?, ?, ?, ?)"
	statement, e := vs.database.Prepare(query)
	if e != nil {
		return e
	}

	if _, e = vs.database.Exec("truncate table tb_porn_index"); e != nil {
		return e
	}

	for i := range videos {
		_, e := statement.Exec(
			videos[i].EnName,
			videos[i].JpName,
			videos[i].Episode,
			videos[i].ImagePath,
			videos[i].VideoPath,
		)
		if e != nil {
			return e
		}
	}

	if e = statement.Close(); e != nil {
		return e
	}
	return nil
}
