package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	// properties, err := utils.LoadPropertiesJSON()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// db := transport.NewDynamoDBClient()

	// for _, property := range properties {
	// 	if property.Name != "" {
	// 		property.PropertyId = property.Name
	// 		property.CellId = utils.ToCell(property.Coordinates).String()

	// 		_, err := db.PutItem("properties", property)
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		fmt.Println(fmt.Sprintf("Inserted %s : ", property.PropertyId))
	// 	}
	// }

	// db := transport.NewDynamoDBClient()

	err := http.ListenAndServe(":4000", mux)
	if err != nil {
		log.Fatal(err)
	}
}
