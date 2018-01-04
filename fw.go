package main

import (
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "time"
    "io/ioutil"
    "encoding/json"
    "encoding/base32"
    "github.com/dgryski/dgoogauth"
    "crypto/rand"
)

type Message struct {
    Token   string  `json:"token"`
}

// List of authenticated IPs
var whitelist = make(map[string]int64)

var otpc *dgoogauth.OTPConfig

func purgelist() {
    // Update list
    for key := range whitelist {
        if whitelist[key] < time.Now().UnixNano() {
            delete(whitelist, key)
        }
    }
}

func authenticate(w http.ResponseWriter, req *http.Request) {
    // Read data from request
    var u Message
    if req.Body == nil {
        http.Error(w, "", 404)
        return
    }
    err := json.NewDecoder(req.Body).Decode(&u)
    if err != nil {
        http.Error(w, "", 404)
        return
    }

    val, err := otpc.Authenticate(u.Token)
    if err != nil {
        http.Error(w, "", 404)
        return
    }
    if val {
        ip, _, _ := net.SplitHostPort(req.RemoteAddr)
        whitelist[ip] = time.Now().UnixNano() + 1000000000 * 60 * 60 * 24 * 7
        go purgelist()
        w.Write([]byte("{\"authenticated\": true}\n"))
    } else {
        w.Write([]byte("{\"authenticated\": false}\n"))
    }
}

func authserv() {
    http.HandleFunc("/auth", authenticate)
    err := http.ListenAndServeTLS(":8000", "server.crt", "server.key", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

// Forwards connections, even TLS
func forward(conn net.Conn) {
    client, err := net.Dial("tcp", os.Args[2])
    if err != nil {
        log.Println("Error: dial failed: %v", err)
    }
    // Extract IP and match it
    ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
    registered, ok := whitelist[ip]
    if (!ok || registered < time.Now().UnixNano()) {
        // Drop connections
        client.Close()
        conn.Close()
        return
    }
    // Forward data
    go func() {
        defer client.Close()
        defer conn.Close()
        io.Copy(client, conn)
    }()
    go func() {
        defer client.Close()
        defer conn.Close()
        io.Copy(conn, client)
    }()
}

func main() {
    if len(os.Args) != 3 {
        log.Fatalf("Usage %s {from host}:port {to host}:port\n", os.Args[0]);
        return
    }

    data, err := ioutil.ReadFile("token")
    var secretBase32 string
    if err != nil {
        // Generate random secret instead of using the test value above.
        secret := make([]byte, 10)
        _, err := rand.Read(secret)
        if err != nil {
            log.Fatalf("Error", err)
        }
        secretBase32 = base32.StdEncoding.EncodeToString(secret)
        data := []byte(secretBase32)
        err = ioutil.WriteFile("token", data, 0644)
        if err != nil {
            log.Fatalf("Error:", err)
        }
    } else {
        secretBase32 = string(data)
    }

    otpc = &dgoogauth.OTPConfig{
        Secret:      secretBase32,
        WindowSize:  3,
        HotpCounter: 0,
        // UTC:         true,
    }

    go authserv()

    listener, err := net.Listen("tcp", os.Args[1])
    if err != nil {
        log.Fatalf("Error: failed to setup listener: %v", err)
    }

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Fatalf("Error: failed to accept listener: %v", err)
        }
        go forward(conn)
    }
}