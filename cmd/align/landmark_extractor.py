#!/usr/bin/env python3
import sys
import dlib
import cv2
import json

def extract_landmarks(image_path, model_path):
    # Load the image
    image = cv2.imread(image_path)
    if image is None:
        return {"error": "Image not found"}

    # Load the face detector and shape predictor
    detector = dlib.get_frontal_face_detector()
    predictor = dlib.shape_predictor(model_path)

    # Detect faces
    faces = detector(image, 1)
    if len(faces) == 0:
        return {"error": "No faces detected"}

    # Assume the first detected face
    face = faces[0]
    landmarks = predictor(image, face)

    # Extract (x, y) coordinates of the 68 landmarks
    points = [(landmarks.part(i).x, landmarks.part(i).y) for i in range(68)]
    return {"landmarks": points}

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print(json.dumps({"error": "Usage: python landmark_extractor.py <image_path> <model_path>"}))
        sys.exit(1)

    image_path = sys.argv[1]
    model_path = sys.argv[2]
    result = extract_landmarks(image_path, model_path)
    print(json.dumps(result))