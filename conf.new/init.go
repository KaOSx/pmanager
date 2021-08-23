package conf

import (
	"io"
	"os"
	"path"
	"pmanager/log"

	"pmanager/util.new/resource"
)

func loadDefaultConf() {
	f, err := model.Open("model/pmanager.conf")
	if err != nil {
		panic("The embed configuration file doesn’nt exist")
	}
	defer f.Close()
	cnf = newConfiguration(f)
}

func loadCustomConf(cnfPath string) (c *configuration, err error) {
	var f io.Reader
	if f, err = resource.Open(cnfPath); err != nil {
		log.Errorf("Failed to read the configuration file: %s\n", err)
	} else {
		c = newConfiguration(f)
	}
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
	log.Debug = cnf.bool("main.debug")
	log.Init(cnf.string("main.logfile"))
	cnfPath := path.Join(ConfDir, ConfFile)
	var modified bool
	exists := resource.IsFile(cnfPath)
	if exists {
		if customCnf, err := loadCustomConf(cnfPath); err == nil {
			if modified = cnf.fusion(customCnf); modified {
				log.Debug = cnf.bool("main.debug")
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
		f, err := os.Create(cnfPath)
		if err != nil {
			log.Fatalf("Failed to create the configuration file: %s\n", err)
		}
		defer f.Close()
		if err := cnf.writeTo(f); err != nil {
			log.Fatalf("Failed to write the configuration file: %s\n", err)
		}
	}
}
