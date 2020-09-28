package persist

import (
	"crypto/sha256"
	"github.com/jinzhu/gorm"
	"github.com/psy-core/psysswd-vault/config"
	"github.com/psy-core/psysswd-vault/internal/constant"
	"github.com/psy-core/psysswd-vault/internal/util"
	"golang.org/x/crypto/pbkdf2"
	"time"
)

func QueryRecord(conf *config.VaultConfig, masterUserName, masterPassword string, recordNameKeyword string) ([]*DecodedRecord, error) {

	db, err := initialDB(conf.PersistConf.DataFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var datas []*AccountRecord
	if recordNameKeyword == "" {
		err = db.
			Where("user_name = ?", masterUserName).
			Find(&datas).Error
	} else {
		err = db.
			Where("user_name = ?", masterUserName).
			Where("name like ?", "%"+recordNameKeyword+"%").
			Find(&datas).Error
	}

	resultRecord := make([]*DecodedRecord, 0, len(datas))
	for _, data := range datas {

		enKey := pbkdf2.Key([]byte(masterPassword), data.Salt, constant.Pbkdf2Iter, 32, sha256.New)
		plainBytes, err := util.AesDecrypt(data.LoginPasswordEn, enKey)
		if err != nil {
			return nil, err
		}

		resultRecord = append(resultRecord, &DecodedRecord{
			Name:          data.Name,
			Description:   data.Description,
			LoginName:     data.LoginName,
			LoginPassword: string(plainBytes),
			ExtraMessage:  data.ExtraMessage,
			CreateTime:    data.CreateTime,
			UpdateTime:    data.UpdateTime,
		})
	}
	return resultRecord, err
}

func ModifyRecord(conf *config.VaultConfig, masterUserName, masterPassword string, newData *DecodedRecord) error {

	db, err := initialDB(conf.PersistConf.DataFile)
	if err != nil {
		return err
	}
	defer db.Close()

	var oldData AccountRecord
	err = db.
		Where("user_name = ?", masterUserName).
		Where("name=?", newData.Name).
		First(&oldData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {

		salt, err := util.RandSalt()
		if err != nil {
			return err
		}
		keyEn := pbkdf2.Key([]byte(masterPassword), salt, constant.Pbkdf2Iter, 32, sha256.New)
		encrypted, err := util.AesEncrypt([]byte(newData.LoginPassword), keyEn)
		if err != nil {
			return err
		}

		saveData := AccountRecord{
			UserName:        masterUserName,
			Name:            newData.Name,
			Description:     newData.Description,
			LoginName:       newData.LoginName,
			Salt:            salt,
			LoginPasswordEn: encrypted,
			ExtraMessage:    newData.ExtraMessage,
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		}
		return db.Save(&saveData).Error
	}

	keyEn := pbkdf2.Key([]byte(masterPassword), oldData.Salt, constant.Pbkdf2Iter, 32, sha256.New)
	encrypted, err := util.AesEncrypt([]byte(newData.LoginPassword), keyEn)
	if err != nil {
		return err
	}
	oldData.Description = newData.Description
	oldData.LoginName = newData.LoginName
	oldData.LoginPasswordEn = encrypted
	oldData.ExtraMessage = newData.ExtraMessage
	oldData.UpdateTime = time.Now()
	return db.Save(&oldData).Error
}
