//package main
//
//import (
//	"bufio"
//	"io"
//	"log"
//	"net/http"
//	"net/url"
//	"os"
//	"sync"
//	"time"
//)
//var wg sync.WaitGroup
//
//// 测试并发，随机出100w个邮箱和手机号
////var channel chan struct {}
//func main() {
//
//	channel = make(chan struct{}, 5000)
//
//	//name := "xutianemng"
//	//password := "xtm20000124"
//	var telephone []string
//	var email []string
//	var sid []string
//
//	//fp1, err := os.Open("/Users/xutianmeng/go/src/TradePlatform/100wtelephones")
//	//fp2, err := os.Open("/Users/xutianmeng/go/src/TradePlatform/100wemails")
//	//fp3, err := os.Open("/Users/xutianmeng/go/src/TradePlatform/100wsids")
//	fp1, err := os.Open("/home/ubuntu/workspace/GoLang/src/TradePlatform/100wtelephones")
//	fp2, err := os.Open("/home/ubuntu/workspace/GoLang/src/TradePlatform/100wemails")
//	fp3, err := os.Open("/home/ubuntu/workspace/GoLang/src/TradePlatform/100wsids")
//	//fp1, err := os.Open("/root/workspace/Go/src/TradePlatform/100wtelephones")
//	//fp2, err := os.Open("/root/workspace/Go/src/TradePlatform/100wemails")
//	//fp3, err := os.Open("/root/workspace/Go/src/TradePlatform/100wsids")
//	if err != nil {
//		log.Println(err)
//		return
//	}
//	defer fp1.Close()
//	defer fp2.Close()
//	defer fp3.Close()
//
//	start := 0
//	limit := 100000
//
//	br := bufio.NewReader(fp1)
//	for {
//		a, _, c := br.ReadLine()
//		if c == io.EOF {
//			break
//		}
//		telephone = append(telephone, string(a))
//	}
//	br2 := bufio.NewReader(fp2)
//	for {
//		a, _, c := br2.ReadLine()
//		if c == io.EOF {
//			break
//		}
//		email = append(email, string(a))
//	}
//	br3 := bufio.NewReader(fp3)
//	for {
//		a, _, c := br3.ReadLine()
//		if c == io.EOF {
//			break
//		}
//		sid = append(sid, string(a))
//	}
//
//	st := time.Now()
//	for i := start; i < limit; i++ {
//		wg.Add(1)
//		go httpPostFormRegister("xutianmeng", "xutianmeng", telephone[i], email[i], sid[i])
//		time.Sleep(10 * time.Millisecond)
//		//break
//		//time.Sleep(time.Millisecond * 100)
//		//go httpGetInfo("1522972330@qq.com")
//		//go httpGetInfo(email[i])
//		//if i % 1000 == 0 {
//		//	//time.Sleep(100 * time.Millisecond)
//		//}
//
//		//wg.Add(1)
//		//go httpPostFormRegister(name, password, telephone[i], email[i], sid[i])
//		//go httpGetInfo(email[i])
//		//if i % 800 == 0 {
//		//	time.Sleep(time.Millisecond * 300)
//		//}
//	}
//	wg.Wait()
//	log.Println(time.Since(st))
//	time.Sleep(5 * time.Second)
//}
////func Get(url string) (*http.Response, error) {
////	//new request
////	req, err := http.NewRequest(http.MethodGet, url, nil)
////	if err != nil {
////		log.Println(err)
////		return nil, errors.New("new request is fail ")
////	}
////	//http client
////	client := &http.Client{}
////	//log.Printf("Go %s URL : %s \n",http.MethodGet, req.URL.String())
////	req.Close = true
////	return client.Do(req)
////}
////// 39.105.229.77
////func httpGetInfo (email string) {
////	channel <- struct {} {}
////	resp, err := Get("http://39.105.229.77:8000/api/auth/info/"+email)
////	if err != nil {
////		log.Println(err)
////		return
////	}
////	resp.Body.Close()
////	wg.Done()
////	<- channel
////}
////152.136.180.243
//func httpPostFormRegister(name, password, telephone, useremail, usersid string) {
//	channel <- struct{} {}
//	http.PostForm("http://39.105.229.77:8000/api/auth/register",
//		url.Values{
//			"UserName": {name},
//			"UserPassword": {password},
//			"UserTelephone" : {telephone},
//			"UserEmail" : {useremail},
//			"UserSid" : {usersid},
//
//		})
//	//
//	//if err != nil {
//	//	// handle error
//	//	log.Println("请求失败, err = ", err)
//	//}
//	//
//	//defer resp.Body.Close()
//	//body, err := ioutil.ReadAll(resp.Body)
//	//if err != nil {
//	//	// handle error
//	//	log.Println("接收相应失败, err = ", err)
//	//}
//	//
//	//fmt.Println(string(body))
//	wg.Done()
//
//	<- channel
//}
//
