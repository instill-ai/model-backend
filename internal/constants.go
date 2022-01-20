package utils

type Dictionary map[string]interface{}

var ModelNames = []Dictionary{
	{"id": "c64j7td9481af4asqb9g", "name": "MobileNetV2 (Ensemble)", "description": "Ensemble model for MobileNetV2 trained on ImageNet for image classification"},
	{"id": "c64j81t9481afcrn2o60", "name": "YOLOv4 (Ensemble)", "description": "Ensemble model for YOLOv4 trained on MSCOCO for object detection"},
	{"id": "c64j8259481afijdgmm0", "name": "MobileNetV2 (Inference)", "description": "Inference part of Ensemble model for MobileNetV2 trained on ImageNet for image classification"},
	{"id": "c64j82t9481afoqqqa10", "name": "YOLOv4 (Inference)", "description": "Inference part of Ensemble model for YOLOv4 trained on MSCOCO for object detection"},
	{"id": "c64j83l9481afve7c78g", "name": "YOLOv4 (Post-processing)", "description": "Post-processing part of Ensemble model for YOLOv4 trained on MSCOCO for object detection"},
	{"id": "c64j84d9481ag524juu0", "name": "MobileNetV2 (Pre-processing)", "description": "Pre-processing part of Ensemble model for MobileNetV2 trained on ImageNet for image classification"},
	{"id": "c64j84d9481ag524jv00", "name": "YOLOv4 (Pre-processing)", "description": "Pre-processing part of Ensemble model for YOLOv4 trained on MSCOCO  for object detection"},
	{"id": "m_3v2Yq6ocICEq0LxDdt8dBtl92Yl3QeWA", "name": "Custom-YOLOv4 (Ensemble)", "description": "Ensemble model for Custom-YOLOv4 for object detection"},
	{"id": "c6ms1bd9481ce7tovia0", "name": "Custom-YOLOv4 (Inference)", "description": "Inference part of Ensemble model for Custom-YOLOv4 for object detection"},
	{"id": "c6ms1bd9481ce7toviag", "name": "Custom-YOLOv4 (Post-process)", "description": "Post-processing part of Ensemble model for Custom-YOLOv4 for object detection"},
	{"id": "c6ms1bd9481ce7tovib0", "name": "Custom-YOLOv4 (Pre-processing)", "description": "Pre-processing part of Ensemble model for Custom-YOLOv4 for object detection"},
}
