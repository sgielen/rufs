package shares

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/sgielen/rufs/proto"
)

type hashListener func(circle, remotePath, hash string)

var (
	hashQueue    = make(chan hashRequest, 1000)
	hashCacheMtx sync.Mutex
	hashCache    = map[string]cachedHash{}
	listeners    []hashListener
)

type hashRequest struct {
	local    string
	remote   string
	download *pb.ConnectResponse_ActiveDownload
}

type cachedHash struct {
	hash  string
	mtime time.Time
	size  int64
}

func StartHash(circle, remotePath string) {
}

func RegisterHashListener(callback hashListener) {
	hashCacheMtx.Lock()
	hashCacheMtx.Unlock()
	listeners = append(listeners, callback)
}

func hashWorker() {
	for req := range hashQueue {
		_, err := hashFile(req.local)
		if err != nil {
			log.Printf("Failed to hash %q: %v", req.local, err)
			continue
		}
	}
}

func getFileHashWithStat(fn string) (string, error) {
	var err error
	st, err := os.Stat(fn)
	if err != nil {
		return "", err
	}
	return getFileHash(fn, st), nil
}

func getFileHash(fn string, st os.FileInfo) string {
	hashCacheMtx.Lock()
	defer hashCacheMtx.Unlock()
	h, ok := hashCache[fn]
	if ok && h.mtime == st.ModTime() && h.size == st.Size() {
		return h.hash
	}
	if ok {
		delete(hashCache, fn)
	}
	return ""
}

func hashFile(fn string) (string, error) {
	fh, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	st, err := fh.Stat()
	if err != nil {
		return "", err
	}
	if h := getFileHash(fn, st); h != "" {
		// We already have the latest hash.
		return h, nil
	}
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	hashCacheMtx.Lock()
	hashCache[fn] = cachedHash{
		hash:  hash,
		mtime: st.ModTime(),
		size:  st.Size(),
	}
	hashCacheMtx.Unlock()
	return hash, nil
}
