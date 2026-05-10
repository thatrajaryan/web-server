package common

type HttpRequest struct {
	Header map[string]string
	Body   string
}

type HttpResponse struct {
	Status  int
	Message string
}

type Server struct {
	IpAddress string
	Port      int
}

type Block interface {
	Create(config map[string]interface{}) error
	Connect(target Block) error
	Update(config map[string]interface{}) error
	Delete() error
}