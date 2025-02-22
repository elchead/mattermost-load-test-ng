// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
)

type Cmd struct {
	Msg     string
	Value   string
	Clients []*ssh.Client
}

type DBSettings struct {
	UserName string
	Password string
	DBName   string
	Host     string
	Engine   string
}

// ProvisionURL takes a URL pointing to a file to be provisioned.
// It works on both local files prefixed with file:// or remote files.
// In case of local files, they are uploaded to the server.
func ProvisionURL(client *ssh.Client, url, filename string) error {
	filePrefix := "file://"
	if strings.HasPrefix(url, filePrefix) {
		// upload file from local filesystem
		path := strings.TrimPrefix(url, filePrefix)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("build file %s has to be a regular file", path)
		}
		if out, err := client.UploadFile(path, "/home/ubuntu/"+filename, false); err != nil {
			return fmt.Errorf("error uploading build: %w %s", err, out)
		}
	} else {
		// download build file from URL
		cmd := fmt.Sprintf("wget -O %s %s", filename, url)
		if out, err := client.RunCommand(cmd); err != nil {
			return fmt.Errorf("failed to run cmd %q: %w %s", cmd, err, out)
		}
	}

	return nil
}

// BuildLoadDBDumpCmds returns a slice of commands that, when piped, feed the
// provided DB dump file into the database. Example:
//
//	zcat dbdump.sql
//	mysql/psql connection_details
func BuildLoadDBDumpCmds(dumpFilename string, dbInfo DBSettings) ([]string, error) {
	cmds := []string{
		fmt.Sprintf("zcat %s", dumpFilename),
	}

	var dbCmd string
	switch dbInfo.Engine {
	case "aurora-postgresql":
		dbCmd = fmt.Sprintf("psql 'postgres://%[1]s:%[2]s@%[3]s/%[4]s?sslmode=disable'", dbInfo.UserName, dbInfo.Password, dbInfo.Host, dbInfo.DBName)
	case "aurora-mysql":
		dbCmd = fmt.Sprintf("mysql -h %[1]s -u %[2]s -p%[3]s %[4]s", dbInfo.Host, dbInfo.UserName, dbInfo.Password, dbInfo.DBName)
	default:
		return []string{}, fmt.Errorf("invalid db engine %s", dbInfo.Engine)
	}

	return append(cmds, dbCmd), nil
}
