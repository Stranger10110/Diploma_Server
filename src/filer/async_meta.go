package filer

import (
	s "../main_settings"
	"github.com/gobwas/ws/wsutil"

	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/chrislusf/seaweedfs/weed/pb"
	"github.com/chrislusf/seaweedfs/weed/pb/filer_pb"
	"github.com/chrislusf/seaweedfs/weed/security"
	"github.com/chrislusf/seaweedfs/weed/util"
	"github.com/golang/protobuf/jsonpb"
)

type AsyncMetaSettings struct {
	Target  string        // "pathPrefix", "/", "path to a folder or common prefix for the folders or files on filer")
	Pattern string        // "pattern", "", "full path or just filename pattern, ex: \"/home/?opher\", \"*.pdf\", see https://golang.org/pkg/path/filepath/#Match ")
	Start   time.Duration // "timeAgo", 0, "start time before now. \"300ms\", \"1.5h\" or \"2h45m\". Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"")
}

func AsyncMeta(websocket net.Conn, settings AsyncMetaSettings) bool {
	grpcDialOption := security.LoadClientTLS(util.GetViper(), "grpc.client")

	var filterFunc func(dir, fname string) bool
	if settings.Pattern != "" {
		if strings.Contains(settings.Pattern, "/") {
			println("watch path pattern", settings.Pattern)
			filterFunc = func(dir, fname string) bool {
				matched, err := filepath.Match(settings.Pattern, dir+"/"+fname)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				return matched
			}
		} else {
			println("watch file pattern", settings.Pattern)
			filterFunc = func(dir, fname string) bool {
				matched, err := filepath.Match(settings.Pattern, fname)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				return matched
			}
		}
	}

	shouldPrint := func(resp *filer_pb.SubscribeMetadataResponse) bool {
		if filterFunc == nil {
			return true
		}
		if resp.EventNotification.OldEntry == nil && resp.EventNotification.NewEntry == nil {
			return false
		}
		if resp.EventNotification.OldEntry != nil && filterFunc(resp.Directory, resp.EventNotification.OldEntry.Name) {
			return true
		}
		if resp.EventNotification.NewEntry != nil && filterFunc(resp.EventNotification.NewParentPath, resp.EventNotification.NewEntry.Name) {
			return true
		}
		return false
	}

	jsonpbMarshaler := jsonpb.Marshaler{
		EmitDefaults: false,
	}
	// websocketWriter := wsutil.NewWriterSize(websocket, ws.StateServerSide, ws.OpText, 0)

	eachEntryFunc := func(resp *filer_pb.SubscribeMetadataResponse) error {
		// jsonpbMarshaler.Marshal(os.Stdout, resp)
		// fmt.Println(resp)

		json, err := jsonpbMarshaler.MarshalToString(resp)
		err = wsutil.WriteServerText(websocket, []byte(json))
		return err
	}

	tailErr := pb.WithFilerClient(s.Settings.FilerAddress[:len(s.Settings.FilerAddress)-1], grpcDialOption, func(client filer_pb.SeaweedFilerClient) error {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := client.SubscribeMetadata(ctx, &filer_pb.SubscribeMetadataRequest{
			ClientName: "tail",
			PathPrefix: settings.Target,
			SinceNs:    time.Now().Add(-settings.Start).UnixNano(),
		})
		if err != nil {
			return fmt.Errorf("listen: %v", err)
		}

		for {
			resp, listenErr := stream.Recv()
			if listenErr == io.EOF {
				return nil
			}
			if listenErr != nil {
				return listenErr
			}
			if !shouldPrint(resp) {
				continue
			}
			if err = eachEntryFunc(resp); err != nil {
				return err
			}
		}

	})
	if tailErr != nil {
		fmt.Printf("tail %s: %v\n", s.Settings.FilerAddress, tailErr)
	}

	return true
}

//type EsDocument struct {
//	Dir         string `json:"dir,omitempty"`
//	Name        string `json:"name,omitempty"`
//	IsDirectory bool   `json:"isDir,omitempty"`
//	Size        uint64 `json:"size,omitempty"`
//	Uid         uint32 `json:"uid,omitempty"`
//	Gid         uint32 `json:"gid,omitempty"`
//	UserName    string `json:"userName,omitempty"`
//	Collection  string `json:"collection,omitempty"`
//	Crtime      int64  `json:"crtime,omitempty"`
//	Mtime       int64  `json:"mtime,omitempty"`
//	Mime        string `json:"mime,omitempty"`
//}
//
//func toEsEntry(event *filer_pb.EventNotification) (*EsDocument, string) {
//	entry := event.NewEntry
//	dir, name := event.NewParentPath, entry.Name
//	id := util.Md5String([]byte(util.NewFullPath(dir, name)))
//	esEntry := &EsDocument{
//		Dir:         dir,
//		Name:        name,
//		IsDirectory: entry.IsDirectory,
//		Size:        entry.Attributes.FileSize,
//		Uid:         entry.Attributes.Uid,
//		Gid:         entry.Attributes.Gid,
//		UserName:    entry.Attributes.UserName,
//		Collection:  entry.Attributes.Collection,
//		Crtime:      entry.Attributes.Crtime,
//		Mtime:       entry.Attributes.Mtime,
//		Mime:        entry.Attributes.Mime,
//	}
//	return esEntry, id
//}
