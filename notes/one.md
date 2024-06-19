# May 31 Notes

```golang
infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
```
- The bitwise OR operator (|) combines these two flags, so each log message will include both the date and the time.

```golang
mux.Use(middleware.Recoverer)
```
- Recoverer is a middleware that recovers from panics, logs the panic (and a backtrace), and returns a HTTP 500 (Internal Server Error) status if possible. Recoverer prints a request ID if one is provided.

-   `pathToTemplateFiles` is passed from the perspective of where the binary is created, so it means that the path needs to be passed from the root folder.

-   SCS implements a session management pattern following the OWASP security guidelines. Session data is stored on the server, and a randomly-generated unique session token (or session ID) is communicated to and from the client in a session cookie.
-   `signal.Notify` causes package signal to relay incoming signals to c. If no signals are provided, all incoming signals will be relayed to c. Otherwise, just the provided signals will.

```
ParseForm populates r.Form and r.PostForm.

For all requests, ParseForm parses the raw query from the URL and updates r.Form.

For POST, PUT, and PATCH requests, it also reads the request body, parses it as a form and puts the results into both r.PostForm and r.Form. Request body parameters take precedence over URL query string values in r.Form.

If the request Body's size has not already been limited by [MaxBytesReader], the size is capped at 10MB.

For other HTTP methods, or when the Content-Type is not application/x-www-form-urlencoded, the request Body is not read, and r.PostForm is initialized to a non-nil, empty value.

[Request.ParseMultipartForm] calls ParseForm automatically. ParseForm is idempotent.
```

- When using `scs` (a session management package for Go) to store complex data types like a struct in the session, you need to use the gob package to register the type. This is because session data is serialized (converted to a byte stream) for storage and deserialized (converted back from a byte stream) when retrieved. The `gob` package is used for this serialization and deserialization process. Hence, we need to register custom data types for it to work correctly.

## PACKAGES FOR MAILER

-  `github.com/vanng822/go-premailer/premailer` ->  inline CSS for mails
-  `github.com/xhit/go-simple-mail/v2` ->   mail service


## Notes

- `DKIM Verification for mail` ->   https://www.emailonacid.com/blog/article/email-deliverability/what_is_dkim_everything_you_need_to_know_about_digital_signatures/

##  For shutdown
-  stop the `listenForMail` function after the waitgroup is empty
-  then close the created channels


##  How is the email sent
- smtp client responsible for sending the mail is created
- email is created with the respecive fields
- email is sent using the smtp client

##  Go-Alone Signer
- `Unsign` and `Parse` can be used to get the data but `Unsiqn` can only get the data while `Parse` parses other information like `Timestamp` as well.

##  Middleware
-   The actual handler receives (w, r) and passes to the chain of middlewares which take each of them modify and then send (mw, mr) to the next middleware and finally to the appropriate route handler and then to the appropriate handler.
-   If you attach middleware to the root Chi router, its effects will also apply to any child routers mounted under it. This means that the middleware will process requests before they reach the handlers in the child routers. The middleware chain is constructed in the order middleware is added, so any middleware on the root router will execute before any middleware added to a child router. This allows you to set up global behaviors and checks at the root level that apply to all routes, including those in child routers.

##  Notes
-   if map, array is passed as a variable in struct, add a condition to check for it in case it's nil, it might break otherwise.

