package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-ldap/ldap/v3"
	"github.com/spf13/viper"
)

type user struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type config struct {
	Address string `json:"address"`
	Port    string `json:"port"`
	Fqdn    string `json:"fqdn"`
	BaseDn  string `json:"basedn"`
	Filter  string `json:"filter"`
}

var C config

// const (
// 	BindUsername = "riemann"
// 	BindPassword = "password"
// 	FQDN = "ldap.forumsys.com"
// 	BaseDN = "ou=mathematicians,dc=example,dc=com"
// 	Filter = "(objectClass=*)"
// )

//Connect to LDAP server
func Connect(c *config) (*ldap.Conn, error) {
	fmt.Println(c.Fqdn)
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:389", c.Fqdn))
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Normal Bind and Search
func BindAndSearch(l *ldap.Conn, c *config, u *user) (*ldap.SearchResult, error) {

	l.Bind(u.Name, u.Password)

	searchReq := ldap.NewSearchRequest(
		c.BaseDn,
		ldap.ScopeBaseObject, // you can also use ldap.ScopeWholeSubtree
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		c.Filter,
		[]string{},
		nil,
	)

	result, err := l.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("Search Error: %s", err)
	}

	if len(result.Entries) > 0 {
		return result, nil
	} else {
		return nil, fmt.Errorf("Couldn't fetch search entries")
	}
}

//Reading configuration file config.json
func ReadConfig() (*config, error) {

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&C)
	if err != nil {
		return nil, err
	}

	return &C, nil
}

func Process(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "Sorry! I am can read only POST request")
	case "POST":

		var U user
		err := json.NewDecoder(r.Body).Decode(&U)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		l, err := Connect(&C)
		if err != nil {
			// log.Fatal(err)
			// fmt.Fprint(w, "Error %v", err)
			http.Error(w, "Error", http.StatusBadRequest)
			break
		}
		defer l.Close()

		result, err := BindAndSearch(l, &C, &U)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(U)
		fmt.Println(result.Entries[0])
		responseString := string(result.Entries[0].DN)
		fmt.Fprint(w, responseString)
		fmt.Println(&C)
		fmt.Println("Authed!")

		// fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", r.Body)
		// // name := r.FormValue("name")
		// // password := r.FormValue("address")
		// fmt.Fprintf(w, "Name = %s\n", U.Name)
		// fmt.Fprintf(w, "Password = %s\n", U.Password)

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func base64generate(username string, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func Basic(w http.ResponseWriter, r *http.Request) {

	var U user
	err := json.NewDecoder(r.Body).Decode(&U)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", C.Address, nil)
	req.Header.Add("Authorization", "Basic "+base64generate(U.Name, U.Password))
	_, err = client.Do(req)
	if err != nil {
		return
	}

	if len(req.Cookies()) < 1 {
		fmt.Fprintf(w, "Unauthorized")
	}
	fmt.Fprintf(w, "OK")
}

func main() {
	conf, err := ReadConfig()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(conf)

	http.HandleFunc("/basic", Basic)
	http.HandleFunc("/auth", Process)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
