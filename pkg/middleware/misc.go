package middleware

// // HTTPResponseModifier modifies the http response for the gateway
// func HTTPResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
// 	md, ok := runtime.ServerMetadataFromContext(ctx)
// 	if !ok {
// 		return nil
// 	}

// 	// set http status code
// 	if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
// 		code, err := strconv.Atoi(vals[0])
// 		if err != nil {
// 			return err
// 		}
// 		// delete the headers to not expose any grpc-metadata in http response
// 		delete(md.HeaderMD, "x-http-code")
// 		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
// 		w.WriteHeader(code)
// 	}

// 	return nil
// }

// // ErrorHandler handles the error response for the gateway
// func ErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {

// 	logger, _ := logx.GetZapLogger(ctx)

// 	// return Internal when Marshal failed
// 	const fallback = `{"code": 13, "message": "failed to marshal error message"}`

// 	s := status.Convert(err)
// 	pb := s.Proto()

// 	w.Header().Del("Trailer")
// 	w.Header().Del("Transfer-Encoding")
// 	contentType := marshaler.ContentType(pb)
// 	if contentType == "application/json" {
// 		w.Header().Set("Content-Type", "application/problem+json")
// 	} else {
// 		w.Header().Set("Content-Type", contentType)
// 	}
// 	if s.Code() == codes.Unauthenticated {
// 		w.Header().Set("WWW-Authenticate", s.Message())
// 	}

// 	buf, err := marshaler.Marshal(pb)
// 	if err != nil {
// 		logger.Error(fmt.Sprintf("Failed to marshal error message %q: %v", s, err))
// 		w.WriteHeader(http.StatusInternalServerError)
// 		if _, err := io.WriteString(w, fallback); err != nil {
// 			logger.Error(fmt.Sprintf("Failed to write response: %v", err))
// 		}
// 		return
// 	}
// 	md, ok := runtime.ServerMetadataFromContext(ctx)
// 	if !ok {
// 		logger.Error("Failed to extract ServerMetadata from context")
// 	}
// 	for k, vs := range md.HeaderMD {
// 		if h, ok := func(key string) (string, bool) {
// 			return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, key), true
// 		}(k); ok {
// 			for _, v := range vs {
// 				w.Header().Add(h, v)
// 			}
// 		}
// 	}

// 	// RFC 7230 https://tools.ietf.org/html/rfc7230#section-4.1.2
// 	// Unless the request includes a TE header field indicating "trailers"
// 	// is acceptable, as described in Section 4.3, a server SHOULD NOT
// 	// generate trailer fields that it believes are necessary for the user
// 	// agent to receive.
// 	doForwardTrailers := strings.Contains(strings.ToLower(r.Header.Get("TE")), "trailers")
// 	if doForwardTrailers {
// 		for k := range md.TrailerMD {
// 			tKey := textproto.CanonicalMIMEHeaderKey(fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k))
// 			w.Header().Add("Trailer", tKey)
// 		}
// 		w.Header().Set("Transfer-Encoding", "chunked")
// 	}
// 	var httpStatus int
// 	switch {
// 	case s.Code() == codes.Internal:
// 		if len(s.Details()) > 0 {
// 			switch v := s.Details()[0].(type) {
// 			case *errdetails.ResourceInfo:
// 				switch v.ResourceType {
// 				case "ray":
// 					httpStatus = http.StatusUnprocessableEntity
// 				default:
// 					httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 				}
// 			default:
// 				httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 			}
// 		} else {
// 			httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 		}
// 	case s.Code() == codes.FailedPrecondition:
// 		if len(s.Details()) > 0 {
// 			switch v := s.Details()[0].(type) {
// 			case *errdetails.PreconditionFailure:
// 				switch v.Violations[0].Type {
// 				case "UPDATE", "DELETE", "STATE", "RENAME":
// 					httpStatus = http.StatusUnprocessableEntity
// 				case "MAX BATCH SIZE LIMITATION":
// 					httpStatus = http.StatusBadRequest
// 				}
// 			default:
// 				httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 			}
// 		} else {
// 			httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 		}
// 	default:
// 		httpStatus = runtime.HTTPStatusFromCode(s.Code())
// 	}
// 	w.WriteHeader(httpStatus)
// 	if _, err := w.Write(buf); err != nil {
// 		logger.Error(fmt.Sprintf("Failed to write response: %v", err))
// 	}
// 	if doForwardTrailers {
// 		for k, vs := range md.TrailerMD {
// 			tKey := fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k)
// 			for _, v := range vs {
// 				w.Header().Add(tKey, v)
// 			}
// 		}
// 	}

// }

// // CustomMatcher matches the headers for the gateway
// func CustomMatcher(key string) (string, bool) {
// 	if strings.HasPrefix(strings.ToLower(key), "jwt-") {
// 		return key, true
// 	}
// 	if strings.HasPrefix(strings.ToLower(key), "instill-") {
// 		return key, true
// 	}

// 	switch strings.ToLower(key) {
// 	case "request-id":
// 		return key, true
// 	case "traceparent", "tracestate":
// 		return key, true
// 	default:
// 		return runtime.DefaultHeaderMatcher(key)
// 	}
// }

// // InjectOwnerToContext injects the owner to the context for initializing models
// func InjectOwnerToContext(ctx context.Context, owner *mgmtpb.User) context.Context {
// 	ctx = metadata.AppendToOutgoingContext(ctx, constant.HeaderAuthTypeKey, "user")
// 	ctx = metadata.AppendToOutgoingContext(ctx, constant.HeaderUserUIDKey, owner.GetUid())
// 	return ctx
// }
