package common

type HttpRequest struct {
	Header map[string]string
	Body string
}

type HttpResponse struct {
	Status int
	Message string
}