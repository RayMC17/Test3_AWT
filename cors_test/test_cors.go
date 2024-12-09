// package main

// import (
// 	"flag"
// 	"log"
// 	"net/http"
// )

// // create a simple HTML page with some JS added. Obviously in a professional
// // setting, we would have the JS code in a script file
// const html = `
// <!DOCTYPE html>
// <html lang="en">
// <head>
//     <meta charset="UTF-8">
// </head>

// <body>
//     <h1>Appletree CORS</h1>
//     <div id="output"></div>
//     <script>
//          document.addEventListener('DOMContentLoaded', function() {
//          fetch("http://localhost:4000/api/v1/healthcheck")
//     .then(response => response.text())
//     .then(text => {
//         document.getElementById("output").innerHTML = text;
//     })
//     .catch(err => {
//         document.getElementById("output").innerHTML = "Error: " + err.message;
//     });
// });
//   </script>
//   </body>
// </html>`

// // A very simple HTTP server
// func main() {
// 	addr := flag.String("addr", ":9000", "Server address")
// 	flag.Parse()

// 	log.Printf("starting server on %s", *addr)
// 	err := http.ListenAndServe(*addr,
// 		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Write([]byte(html))
// 		}))
// 	log.Fatal(err)
// }

package main

import (
    "flag"
    "log"
    "net/http"
)
// We will access the POST /v1/tokens/authentication endpoint
const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Appletree Preflight CORS</h1>
    <div id="output"></div>
    <script>
         document.addEventListener('DOMContentLoaded', function() {
         fetch("http://localhost:4000/api/v1/authenticate/token", {
           method: "POST",
           headers: {
                    'Content-Type': 'application/json'
                    },
           body: JSON.stringify({
                    email: 'alice@example.com',
                    password: 'securepassword123'
                 })
           }).then( function(response) {

            response.text().then(function (text) {
                document.getElementById("output").innerHTML = text;
             });
           },
           function(err) {
             document.getElementById("output").innerHTML = err;
           }
        );
     });
</script>
</body>
</html>`

// A very simple HTTP server
func main() {
    addr := flag.String("addr", ":9000", "Server address")
    flag.Parse()

    log.Printf("starting server on %s", *addr)


    err := http.ListenAndServe(*addr, 
           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              w.Write([]byte(html))
       }))
    log.Fatal(err)
}

