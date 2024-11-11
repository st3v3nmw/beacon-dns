package lists

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	Minio      *minio.Client
	BucketName string
)

func NewMinioClient(endpoint, keyId, key, region string) (err error) {
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(keyId, key, ""),
		Region: region,
		Secure: true,
	}
	Minio, err = minio.New(endpoint, opts)
	if err != nil {
		return err
	}

	return nil
}

func statObject(filename string) (*minio.ObjectInfo, error) {
	reader, err := Minio.GetObject(
		context.Background(),
		BucketName,
		filename,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	info, err := reader.Stat()
	return &info, err
}

type Action string

const (
	ActionAllow Action = "allow"
	ActionBlock Action = "block"
)

type Category string

const (
	CategoryAds            Category = "ads"             // ads, trackers
	CategoryMalware        Category = "malware"         // malware, ransomware, phishing, cryptojacking, stalkerware
	CategoryAdult          Category = "adult"           // adult content
	CategoryDating         Category = "dating"          // dating
	CategorySocialMedia    Category = "social-media"    // social media
	CategoryVideoStreaming Category = "video-streaming" // video streaming platforms
	CategoryGambling       Category = "gambling"        // gambling
	CategoryGaming         Category = "gaming"          // gaming
	CategoryPiracy         Category = "piracy"          // piracy, torrents
	CategoryDrugs          Category = "drugs"           // drugs
)

// Lists

type List struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Action      Action    `json:"action"`
	Category    Category  `json:"category"`
	LastSync    time.Time `json:"last_sync"`
	Domains     []string  `json:"domains"`
}

func newFromSource(source Source) (*List, error) {
	resp, err := http.Get(source.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	l := &List{
		Name:        source.Name,
		Description: source.Description,
		URL:         source.URL,
		Action:      source.Action,
		Category:    source.Category,
		LastSync:    time.Now().UTC(),
		Domains:     parseDomains(body, source.Format),
	}
	return l, nil
}

func newFromFs(filename string) (*List, error) {
	path := DataDir + filename
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var list List
	err = json.Unmarshal(data, &list)
	return &list, err
}

func newFromBucket(filename string) (*List, error) {
	reader, err := Minio.GetObject(
		context.Background(),
		BucketName,
		filename,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if _, err := reader.Stat(); err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}

	var list List
	err = json.Unmarshal(data, &list)
	list.LastSync = time.Now()
	return &list, err
}

func (l *List) filename() string {
	return fmt.Sprintf("%s.json", l.Name)
}

func (l *List) existsOnFs() bool {
	path := DataDir + l.filename()
	_, err := os.Stat(path)
	return err == nil
}

func (l *List) saveInFs() error {
	data, err := json.MarshalIndent(l, "", " ")
	if err != nil {
		return err
	}

	path := DataDir + l.filename()
	return os.WriteFile(path, data, 0755)
}

func (l *List) saveInBucket() error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}

	opts := minio.PutObjectOptions{ContentType: "application/json"}
	_, err = Minio.PutObject(
		context.Background(),
		BucketName,
		l.filename(),
		bytes.NewReader(data),
		int64(len(data)),
		opts,
	)
	return err
}
