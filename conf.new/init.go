package conf

import (
	"os"
	"path"
	"pmanager/log"
)

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func loadDefaultConf() {
	f, err := model.Open("model/pmanager.conf")
	if err != nil {
		panic("The embed configuration file doesn’nt exist")
	}
	defer f.Close()
	cnf = newConfiguration(f)
}

func loadCustomConf(cnfPath string) (c *configuration, err error) {
	var f *os.File
	if f, err = os.Open(cnfPath); err != nil {
		log.Errorf("Failed to read the configuration file: %s\n", err)
		return
	}
	defer f.Close()
	c = newConfiguration(f)
	return
}

func saveConf(cnfPath string) {
	f, err := os.Create(cnfPath)
	if err != nil {
		log.Errorf("Failed to save the configuration file: %s\n", err)
		return
	}
	defer f.Close()
	if err := cnf.writeTo(f); err == nil {
		log.Printf("Configuration file saved to %s\n", cnfPath)
	} else {
		log.Errorf("Failed to save the configuration file: %s\n", err)
	}
}

func init() {
	loadDefaultConf()
	log.Init(cnf.string("main.logfile"))
	cnfPath := path.Join(ConfDir, ConfFile)
	var modified bool
	exists := FileExists(cnfPath)
	if !exists {
		if customCnf, err := loadCustomConf(cnfPath); err == nil {
			if modified = cnf.fusion(customCnf); modified {
				log.Init(cnf.string("main.logfile"))
			}
		} else {
			modified = true
		}
	}
	if !exists || modified {
		if exists {
			savePath := cnfPath + ".save"
			if err := os.Rename(cnfPath, savePath); err == nil {
				log.Printf("Old configuration saved in %s\n", savePath)
			} else {
				log.Warnln("Failed to move the old configuration")
			}
		}
	}
	Debug = cnf.bool("main.debug")
}
