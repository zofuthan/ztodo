package zweb

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	StaticDir    string
	Addr         string
	Port         int
	CookieSecret string
	RecoverPanic bool
}

// Server represents a zweb.go server.
type Server struct {
	Config *ServerConfig
	routes []route
	Logger *log.Logger
	Env    map[string]interface{}
	//save the listener so it can be closed
	l net.Listener
}

func NewServer() *Server {
	return &Server{
		Config: Config,
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Env:    map[string]interface{}{},
	}
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
	}

	if s.Logger == nil {
		s.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	}
}

type route struct {
	r           string
	cr          *regexp.Regexp
	method      string
	handler     reflect.Value
	httpHandler http.Handler
}

func (s *Server) addRoute(r string, method string, handler interface{}) {
	cr, err := regexp.Compile(r)
	if err != nil {
		s.Logger.Printf("Error in route regex %q\n", r)
		return
	}

	switch handler.(type) {
	case http.Handler:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, httpHandler: handler.(http.Handler)})
	case reflect.Value:
		fv := handler.(reflect.Value)
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: fv})
	default:
		fv := reflect.ValueOf(handler)
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: fv})
	}
}

// ServeHTTP is the interface method for Go's http server package
func (s *Server) ServeHTTP(c http.ResponseWriter, req *http.Request) {
	s.Process(c, req)
}

// Process invokes the routing system for server s
func (s *Server) Process(c http.ResponseWriter, req *http.Request) {
	route := s.routeHandler(req, c)
	if route != nil {
		route.httpHandler.ServeHTTP(c, req)
	}
}

// Get adds a handler for the 'GET' http method for server s.
func (s *Server) Get(route string, handler interface{}) {
	s.addRoute(route, "GET", handler)
}

// Post adds a handler for the 'POST' http method for server s.
func (s *Server) Post(route string, handler interface{}) {
	s.addRoute(route, "POST", handler)
}

// Put adds a handler for the 'PUT' http method for server s.
func (s *Server) Put(route string, handler interface{}) {
	s.addRoute(route, "PUT", handler)
}

// Delete adds a handler for the 'DELETE' http method for server s.
func (s *Server) Delete(route string, handler interface{}) {
	s.addRoute(route, "DELETE", handler)
}

// Match adds a handler for an arbitrary http method for server s.
func (s *Server) Match(method string, route string, handler interface{}) {
	s.addRoute(route, method, handler)
}

//Adds a custom handler. Only for webserver mode. Will have no effect when running as FCGI or SCGI.
func (s *Server) Handler(route string, method string, httpHandler http.Handler) {
	s.addRoute(route, method, httpHandler)
}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	s.initServer()

	mux := http.NewServeMux()

	mux.Handle("/", s)

	s.Logger.Printf("zweb.go serving %s\n", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// Close stops server s.
func (s *Server) Close() {
	if s.l != nil {
		s.l.Close()
	}
}

// safelyCall invokes `function` in recover block
func (s *Server) safelyCall(function reflect.Value, args []reflect.Value) (resp []reflect.Value, e interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if !s.Config.RecoverPanic {
				// go back to panic
				panic(err)
			} else {
				e = err
				resp = nil
				s.Logger.Println("Handler crashed with error", err)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					s.Logger.Println(file, line)
				}
			}
		}
	}()
	return function.Call(args), nil
}

// requiresContext determines whether 'handlerType' contains
// an argument to 'web.Ctx' as its first argument
func requiresContext(handlerType reflect.Type) bool {
	//if the method doesn't take arguments, no
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, no
	a0 := handlerType.In(0)
	if a0.Kind() != reflect.Ptr {
		return false
	}
	//if the first argument is a context, yes
	if a0.Elem() == contextType {
		return true
	}

	return false
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
// It checks the following directories for the file, in order:
// 1) Config.StaticDir
// 2) The 'static' directory in the parent directory of the executable.
// 3) The 'static' directory in the current working directory
func (s *Server) tryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	//try to serve a static file
	if s.Config.StaticDir != "" {
		staticFile := path.Join(s.Config.StaticDir, name)
		if fileExists(staticFile) {
			http.ServeFile(w, req, staticFile)
			return true
		}
	} else {
		for _, staticDir := range defaultStaticDirs {
			staticFile := path.Join(staticDir, name)
			if fileExists(staticFile) {
				http.ServeFile(w, req, staticFile)
				return true
			}
		}
	}
	return false
}

func (s *Server) logRequest(ctx Context, sTime time.Time) {
	//log the request
	var logEntry bytes.Buffer
	req := ctx.Request
	requestPath := req.URL.Path

	duration := time.Now().Sub(sTime)
	var client string

	// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
	// documentation at http://golang.org/pkg/net/http/#Request
	pos := strings.LastIndex(req.RemoteAddr, ":")
	if pos > 0 {
		client = req.RemoteAddr[0:pos]
	} else {
		client = req.RemoteAddr
	}

	fmt.Fprintf(&logEntry, "%s - \033[32;1m %s %s\033[0m - %v", client, req.Method, requestPath, duration)

	if len(ctx.Params) > 0 {
		fmt.Fprintf(&logEntry, " - \033[37;1mParams: %v\033[0m\n", ctx.Params)
	}

	ctx.Server.Logger.Print(logEntry.String())

}

// the main route handler in zweb.go
// Tries to handle the given request.
// Finds the route matching the request, and execute the callback associated
// with it.  In case of custom http handlers, this function returns an "unused"
// route. The caller is then responsible for calling the httpHandler associated
// with the returned route.
func (s *Server) routeHandler(req *http.Request, w http.ResponseWriter) (unused *route) {
	requestPath := req.URL.Path
	ctx := Context{req, map[string]string{}, s, w}

	//set some default headers
	ctx.SetHeader("Server", "zweb.go", true)
	tm := time.Now().UTC()

	//ignore errors from ParseForm because it's usually harmless.
	req.ParseForm()
	if len(req.Form) > 0 {
		for k, v := range req.Form {
			ctx.Params[k] = v[0]
		}
	}

	defer s.logRequest(ctx, tm)

	ctx.SetHeader("Date", webTime(tm), true)

	if req.Method == "GET" || req.Method == "HEAD" {
		if s.tryServingFile(requestPath, req, w) {
			return
		}
	}

	//Set the default content-type
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)

	for i := 0; i < len(s.routes); i++ {
		route := s.routes[i]
		cr := route.cr
		//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
		if req.Method != route.method && !(req.Method == "HEAD" && route.method == "GET") {
			continue
		}

		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)

		if len(match[0]) != len(requestPath) {
			continue
		}

		if route.httpHandler != nil {
			unused = &route
			// We can not handle custom http handlers here, give back to the caller.
			return
		}

		var args []reflect.Value
		handlerType := route.handler.Type()
		if requiresContext(handlerType) {
			args = append(args, reflect.ValueOf(&ctx))
		}
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}

		ret, err := s.safelyCall(route.handler, args)
		if err != nil {
			//there was an error or panic while calling the handler
			ctx.Abort(500, "Server Error")
		}
		if len(ret) == 0 {
			return
		}

		sval := ret[0]

		var content []byte

		if sval.Kind() == reflect.String {
			content = []byte(sval.String())
		} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
			content = sval.Interface().([]byte)
		}
		ctx.SetHeader("Content-Length", strconv.Itoa(len(content)), true)
		_, err = ctx.ResponseWriter.Write(content)
		if err != nil {
			ctx.Server.Logger.Println("Error during write: ", err)
		}
		return
	}

	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if s.tryServingFile(path.Join(requestPath, "index.html"), req, w) {
			return
		} else if s.tryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			return
		}
	}
	ctx.Abort(404, "Page not found")
	return
}

// SetLogger sets the logger for server s
func (s *Server) SetLogger(logger *log.Logger) {
	s.Logger = logger
}
