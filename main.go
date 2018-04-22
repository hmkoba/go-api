package main

import (
	"log"
	"os"
	"net/http"
	"time"
	"fmt"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
)

import "github.com/timjacobi/go-couchdb"

type Visitor struct {
    Name      string    `json:"name"`
}

type Visitors []Visitor

type alldocsResult struct {
	TotalRows int `json:"total_rows"`
	Offset    int
	Rows      []map[string]interface{}
}

func main() {
	r := gin.Default()

	r.StaticFile("/", "./static/index.html")
	r.Static("/static", "./static")

	var dbName = "mydb"

	//When running locally, get credentials from .env file.
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file does not exist")
	}
  cloudantUrl := os.Getenv("CLOUDANT_URL")

	appEnv, _ := cfenv.Current()
  if(appEnv!=nil){
    cloudantService, _ := appEnv.Services.WithLabel("cloudantNoSQLDB")
    if(len(cloudantService)>0){
      cloudantUrl = cloudantService[0].Credentials["url"].(string)
    }
  }

  cloudant, err := couchdb.NewClient(cloudantUrl, nil)
	if err != nil {
		log.Println("Can not connect to Cloudant database")
	}

  //ensure db exists
  //if the db exists the db will be returned anyway
  cloudant.CreateDB(dbName)

	/* Endpoint to greet and add a new visitor to database.
	* Send a POST request to http://localhost:8080/api/visitors with body
	* {
	* 	"name": "Bob"
	* }
	*/
	r.POST("/api/visitors", func(c *gin.Context) {
		var visitor Visitor
    if c.BindJSON(&visitor) == nil {
      cloudant.DB(dbName).Post(visitor)
			c.String(200, "Hello "+visitor.Name)
		}
	})

	/**
	 * Endpoint to get a JSON array of all the visitors in the database
	 * REST API example:
	 * <code>
	 * GET http://localhost:8080/api/visitors
	 * </code>
	 *
	 * Response:
	 * [ "Bob", "Jane" ]
	 * @return An array of all the visitor names
	 */
  r.GET("/api/visitors", func(c *gin.Context) {
    var result alldocsResult
    if cloudantUrl == "" {
      c.JSON(200, gin.H{})
      return
    }
    err := cloudant.DB(dbName).AllDocs(&result, couchdb.Options{"include_docs": true})
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch docs"})
    } else {
      c.JSON(200, result.Rows)
    }
  })


	r.GET("/api/line/test", func(c *gin.Context) {
    c.String(200, "line test")
  })

  r.POST("/api/line/webhook", func(c *gin.Context) {
		client := &http.Client{Timeout: time.Duration(15 * time.Second)}
		 bot, err := linebot.New("<Channel Secret>", "<Channel Access Token>", linebot.WithHTTPClient(client))
		 if err != nil {
				 fmt.Println(err)
				 return
		 }
		 received, err := bot.ParseRequest(c.Request)

		 for _, event := range received {
				 if event.Type == linebot.EventTypeMessage {
				     source := event.Source
						 switch source.Type {
						 case linebot.EventSourceTypeUser:
							 log.Print("userId:" + source.UserID);

						 case linebot.EventSourceTypeRoom:
							 log.Print("userId:" + source.UserID + "  groupId:" +source.GroupID);

						 case linebot.EventSourceTypeGroup:
							 log.Print("userId:" + source.UserID + "  roomId:" + source.RoomID);
						 }
				 }
		 }
 })

	//When running on Bluemix, get the PORT from the environment variable.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8070" //Local
	}
	r.Run(":" + port)
}
