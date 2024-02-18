package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	goinsta "github.com/Davincible/goinsta/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	REQUESTS_COLLECTION = "Requests"
	LOGIN_COLLECTION    = "Login"
)

var (
	fsClient     *firestore.Client
	fsClientOnce sync.Once

	insta *goinsta.Instagram
)

type LoginData struct {
	Data          string
	OriginalLogin time.Time
}

type ListRequestResponse struct {
	Response Response `json:"response"`
}

type RequestDoc struct {
	CreatedDate time.Time
}

type Versions struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
}
type PrimaryAttachment struct {
	ID          int      `json:"id"`
	Extension   string   `json:"extension"`
	ContentType string   `json:"content_type"`
	URL         string   `json:"url"`
	Versions    Versions `json:"versions"`
}

type RequestWrapper struct {
	Request Request `json:"request"`
}

type Request struct {
	PrimaryAttachment PrimaryAttachment `json:"primary_attachment"`
	ID                int               `json:"id"`
	ImageThumbnail    string            `json:"image_thumbnail"`
	Title             string            `json:"title"`
	Description       string            `json:"description"`
	Status            string            `json:"status"`
	Address           string            `json:"address"`
	Location          string            `json:"location"`
	Zipcode           any               `json:"zipcode"`
	ForeignID         string            `json:"foreign_id"`
	DateCreated       int               `json:"date_created"`
	CountComments     int               `json:"count_comments"`
	CountFollowers    int               `json:"count_followers"`
	CountSupporters   int               `json:"count_supporters"`
	Lat               float64           `json:"lat"`
	Lon               float64           `json:"lon"`
	UserFollows       int               `json:"user_follows"`
	UserComments      int               `json:"user_comments"`
	UserRequest       int               `json:"user_request"`
	Rank              string            `json:"rank"`
	User              string            `json:"user"`
	IGPostedAt        time.Time
}

type Status struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Code        int    `json:"code"`
	CodeMessage string `json:"code_message"`
}
type Response struct {
	Requests  []RequestWrapper `json:"requests"`
	Count     string           `json:"count"`
	Benchmark float64          `json:"benchmark"`
	Status    Status           `json:"status"`
}

func (r *Request) CreationDate() time.Time {
	return time.Unix(int64(r.DateCreated), 0)
}

func (r *Request) ToDoc() {

}

// request handler
func HandleProcess(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}

	v := url.Values{}

	v.Set("limit", "100")
	v.Set("client_id", "242")
	v.Set("device", "iframe")

	req, err := http.NewRequest(http.MethodGet, "https://vc0.publicstuff.com/api/2.0/requests_list?"+v.Encode(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bb, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var resp ListRequestResponse
	if err := json.Unmarshal(bb, &resp); err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = initInstaClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, v := range resp.Response.Requests {
		if !shouldPost(v.Request) {
			continue
		}

		err := postIG(&v.Request)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		v.Request.IGPostedAt = time.Now()
		err = saveRequest(v.Request)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

	}
}

func initInstaClient() error {
	ld, err := importLogin()
	//not found in db
	if status.Code(err) == codes.NotFound || (ld != nil && ld.OriginalLogin.Before(time.Now().Add(-time.Hour*12))) {
		if err := freshLogin(); err != nil {
			return err
		}
	}
	//some other err
	if err != nil {
		return err
	}

	insta, err = goinsta.ImportFromBase64String(ld.Data)
	if err != nil {
		return err
	}

	return nil
}

func freshLogin() error {
	insta = goinsta.New(os.Getenv("IG_USER"), os.Getenv("IG_PASS"))
	if err := insta.Login(); err != nil {
		return err
	}
	defer exportLogin(insta)
	return nil
}

func postIG(r *Request) error {
	resp, err := http.Get(r.PrimaryAttachment.Versions.Large)
	if err != nil {
		return err
	}

	var caption string
	if r.Description != "" && r.Description != "undefined" {
		caption = fmt.Sprintf("%s: \"%s\"", r.Title, r.Description)
	}

	captionedImage, err := ProcessImage(resp.Body, caption)
	if err != nil {
		return err
	}

	upload := &goinsta.UploadOptions{
		File:    captionedImage,
		IsStory: true,
	}

	_, err = insta.Upload(upload)

	if err != nil {
		return err
	}
	fmt.Println(r.Description)
	return nil
}

func shouldPost(r Request) bool {
	if r.PrimaryAttachment.Versions.Large == "" {
		fmt.Fprintln(os.Stderr, "skipping due to no image")
		return false
	}

	if r.Description == "" || r.Description == "undefined" {
		fmt.Fprintln(os.Stderr, "skipping due to no description")
		return false
	}

	if strings.Contains(strings.ToLower(r.Title), "graffiti") {
		fmt.Fprintln(os.Stderr, "skipping due to graffiti")
		return false
	}

	//check if already posted

	doc, err := client().Collection(REQUESTS_COLLECTION).Doc(fmt.Sprintf("%d", r.ID)).Get(context.Background())
	if status.Code(err) == codes.NotFound {
		return true
	}

	if doc.Exists() {
		return false
	}

	return true
}

func client() *firestore.Client {
	fsClientOnce.Do(func() {
		ctx := context.Background()
		client, err := firestore.NewClient(ctx, "philly311")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fsClient = client
	})

	return fsClient
}

func saveRequest(r Request) error {
	requestsColl := client().Collection("Requests")

	_, err := requestsColl.Doc(fmt.Sprintf("%d", r.ID)).Set(context.Background(), r)
	if err != nil {
		return err
	}

	return nil
}

func importLogin() (*LoginData, error) {
	ds, err := client().Collection(LOGIN_COLLECTION).Doc("0").Get(context.Background())
	if err != nil {
		return nil, err
	}

	var loginData LoginData
	if err := ds.DataTo(&loginData); err != nil {
		return nil, err
	}

	return &loginData, nil
}

func exportLogin(insta *goinsta.Instagram) error {
	b64str, err := insta.ExportAsBase64String()
	if err != nil {
		return err
	}

	_, err = client().Collection(LOGIN_COLLECTION).Doc("0").Set(context.Background(), LoginData{Data: b64str, OriginalLogin: time.Now()})
	if err != nil {
		return err
	}

	return nil
}
