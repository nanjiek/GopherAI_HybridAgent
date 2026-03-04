package image

import (
	"GopherAI/common/image"
	"GopherAI/config"
	"io"
	"log"
	"mime/multipart"
)

func RecognizeImage(file *multipart.FileHeader) (string, error) {
	conf := config.GetConfig()
	modelPath := conf.ImageConfig.ModelPath
	labelPath := conf.ImageConfig.LabelPath
	inputH, inputW := conf.ImageConfig.InputH, conf.ImageConfig.InputW

	recognizer, err := image.NewImageRecognizer(modelPath, labelPath, inputH, inputW)
	if err != nil {
		log.Println("NewImageRecognizer fail err is : ", err)
		return "", err
	}
	defer recognizer.Close()

	src, err := file.Open()
	if err != nil {
		log.Println("file open fail err is : ", err)
		return "", err
	}
	defer src.Close()

	buf, err := io.ReadAll(src)
	if err != nil {
		log.Println("io.ReadAll fail err is : ", err)
		return "", err
	}

	return recognizer.PredictFromBuffer(buf)
}
