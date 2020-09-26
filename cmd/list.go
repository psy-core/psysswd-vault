package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/psy-core/psysswd-vault/config"
	"github.com/psy-core/psysswd-vault/internal/constant"
	"github.com/psy-core/psysswd-vault/internal/util"
	"golang.org/x/crypto/pbkdf2"
	"io/ioutil"
	"os"
	"strings"

	"github.com/psy-core/psysswd-vault/internal/auth"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list account info for given username",
	Long:  `list account info for given username`,
	Args:  cobra.NoArgs,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	vaultConf, err := config.InitConf(cmd.Flags().GetString("conf"))
	checkError(err)
	username, password, err := readUsernameAndPassword(cmd, vaultConf)
	checkError(err)

	if !auth.Auth(username, password) {
		fmt.Println("Permission Denied.")
		os.Exit(1)
	}

	indexData, err := ioutil.ReadFile("index.data")
	bodyData, err := ioutil.ReadFile("body.data")

	for i := 0; i < len(indexData); i += 32 {
		var keyOffset int64
		var keyLen int32
		binary.Read(bytes.NewBuffer(indexData[i+20:i+28]), binary.LittleEndian, &keyOffset)
		binary.Read(bytes.NewBuffer(indexData[i+28:i+32]), binary.LittleEndian, &keyLen)
		key := string(bodyData[keyOffset : keyOffset+int64(keyLen)])
		if strings.HasPrefix(key, username) {
			var dataOffset int64
			var dataLen int32
			binary.Read(bytes.NewBuffer(indexData[i+8:i+16]), binary.LittleEndian, &dataOffset)
			binary.Read(bytes.NewBuffer(indexData[i+16:i+20]), binary.LittleEndian, &dataLen)
			passwordEn := bodyData[dataOffset : dataOffset+int64(dataLen)]
			var saltLen int32
			binary.Read(bytes.NewBuffer(passwordEn[:4]), binary.LittleEndian, &saltLen)
			salt := passwordEn[4:4+saltLen]
			passEnKey := pbkdf2.Key([]byte(password), salt, constant.Pbkdf2Iter, 32, sha256.New)
			plainBytes, err := util.AesDecrypt(passwordEn[4+saltLen:], passEnKey)
			checkError(err)
			fmt.Printf("username: %s, password: %s\n", strings.TrimPrefix(key, username), plainBytes)
		}
	}
}
