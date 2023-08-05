# Scripts-go

## What does it do?
For each image-folder, create `.zip` file contains resized `avif`/`webp` version of the images, a `.7z` file contains lossless `jxl` version ones for archiving.

## Requirements
- BatchAVIF:
  - ffmpeg
  - ffprobe
  - aomenc (or modify the script to use your encoder of choice)
- BatchCompress:
  - 7z
- BatchJXL:
  - cjxl
- BatchResize, UpscaleAni:
  - ffmpeg
  - ffprobe
  - [realesrgan-ncnn-vulkan](https://github.com/xinntao/Real-ESRGAN)
