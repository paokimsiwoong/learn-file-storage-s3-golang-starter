package main

// GET /api/thumbnails/{videoID} handler : 제시된 비디오에 해당하는 썸네일 구조체를 글로벌 맵에서 찾아 반환
// func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
// 	videoIDString := r.PathValue("videoID")
// 	videoID, err := uuid.Parse(videoIDString)
// 	if err != nil {
// 		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
// 		return
// 	}

// 	tn, ok := videoThumbnails[videoID]
// 	if !ok {
// 		respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
// 		return
// 	}

// 	w.Header().Set("Content-Type", tn.mediaType)
// 	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(tn.data)))

// 	_, err = w.Write(tn.data)
// 	if err != nil {
// 		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
// 		return
// 	}
// }
