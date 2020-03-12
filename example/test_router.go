package main

import (
	"fmt"

	tarantool "github.com/viciious/go-tarantool"
)

func TestRouter() {
	opts := tarantool.Options{User: "qumomf", Password: "qumomf", UUID: "some_uuid_4"}
	conn, err := tarantool.Connect("127.0.0.1:9304", &opts)
	if err != nil {
		fmt.Println("Connection refused:", err)
	}
	fmt.Println(conn)
	//resp, err := conn.Insert(999, []interface{}{99999, "BB"})
	//if err != nil {
	//	fmt.Println("Error", err)
	//	fmt.Println("Code", resp.Code)
	//}
}

func main() {
	TestRouter()
}
