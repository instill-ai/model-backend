
#!/bin/sh

echo "Downloading conda-pack"
mkdir conda-pack > /dev/null 2>&1
wget https://artifacts.instill.tech/vdp/conda-pack/python-3-8.tar.gz -P ./conda-pack/

echo "Downloading sample-models"
mkdir sample-models > /dev/null 2>&1
wget https://artifacts.instill.tech/vdp/sample-models/yolov4-onnx-cpu.zip -P ./sample-models/

echo "Download sample images"
wget https://artifacts.instill.tech/dog.jpg -P ./sample-models/

echo "Finished!"