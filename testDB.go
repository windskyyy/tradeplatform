package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// 49.233.25.176
var channel chan struct {}
//var client := &http.Client{}
var client = &http.Client{}
var wg sync.WaitGroup
func main() {
	channel = make(chan struct{}, 8000)
	fp, err := os.Open("/home/ubuntu/workspace/GoLang/src/TradePlatform/100wemails")
	if err != nil {
		log.Println(err)
		return
	}
	start := 0
	limit := 50000
	var email []string
	br2 := bufio.NewReader(fp)
	for {
		a, _, c := br2.ReadLine()
		if c == io.EOF {
			break
		}
		email = append(email, string(a))
	}
	fp.Close()

	// 并发
	s := time.Now()
	//for i := start; i < limit; i++ {
	//	wg.Add(1)
	//	go httpGetInfo(email[i])
	//	if i != 0 && i % 10000 == 0 {
	//		fmt.Println("i = ", i)
	//		time.Sleep(1000 * time.Millisecond)
	//		//break
	//	}
	//}

	for i := start; i < limit; i++ {
		wg.Add(1)
		channel <- struct {} {}
		go httpGetInfo(email[i])
		if i != 0 && i % 500 == 0 {
			fmt.Println("i = ", i)
			time.Sleep(50 * time.Millisecond)
		}
	}

	wg.Wait()

	// 串行
	//s := time.Now()
	//for i := start; i < limit; i++ {
	//	wg.Add(1)
	//	httpGetInfo(email[i])
	//}
	fmt.Println(time.Since(s))


	//DB := common.GetDB()
	//defer DB.Close()
	//
	//user := model.User{
	//	UserName: "xutianmeng",
	//	UserEmail: "1522972330@qq.com",
	//	UserTelephone: "15665893057",
	//	UserSid: "20171877138",
	//	UserPassword: "xtm20000124",
	//	UserClass: "BigData",
	//	UserSignature: "i am xutianmeng",
	//}
	//DB.Exec("select * from userInfo")
	//DB.Create(&user)

}
func Get(url string) (*http.Response, error) {
	//new request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("new request is fail ")
	}
	//http client

	//log.Printf("Go %s URL : %s \n",http.MethodGet, req.URL.String())
	req.Close = true
	return client.Do(req)
}
// 39.105.229.77
func httpGetInfo (email string) {
	//channel <- struct {} {}
	resp, err := Get("http://127.0.0.1:8000/api/auth/info/"+email)
	if err != nil {
		log.Println(err)
		return
	}
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	log.Println("body: " , err)
	//}
	//fmt.Println(string(body))
	resp.Body.Close()
	wg.Done()
	<- channel
}

/*
	UserName string `gorm:"type:varchar(100);not null"`
	UserEmail string `gorm:"type:varchar(100);not null;unique"`
	UserTelephone string `gorm:"varchar(15);not null;unique"`
	UserSid string `gorm:"type:varchar(15);not null; unique"`
	UserPassword string `gorm:"type:varchar(15);not null"`
	UserClass string `gorm:"type:varchar(50)"`
*/
