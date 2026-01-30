package main

// import (
// 	"context"
// 	"net"
// 	"strings"

// 	"github.com/byBit-ovo/coral_word/pb"
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"
// )

// type coralWordGrpcServer struct {
// 	pb.UnimplementedCoralWordServiceServer
// }

// func RunGrpcServer(addr string) error {
// 	listener, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		return err
// 	}
// 	stopRegister, err := RegisterGrpcToEtcd(addr)
// 	if err != nil {
// 		_ = listener.Close()
// 		return err
// 	}
// 	defer func() {
// 		_ = stopRegister()
// 	}()
// 	server := grpc.NewServer()
// 	pb.RegisterCoralWordServiceServer(server, &coralWordGrpcServer{})
// 	return server.Serve(listener)
// }

// func (s *coralWordGrpcServer) QueryWord(ctx context.Context, req *pb.WordRequest) (*pb.WordDesc, error) {
// 	if req == nil || strings.TrimSpace(req.Word) == "" {
// 		return nil, status.Error(codes.InvalidArgument, "word is empty")
// 	}
// 	wordDesc, err := QueryWords(req.Word)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "query word error: %v", err)
// 	}
// 	return toPbWordDesc(wordDesc), nil
// }

// func toPbWordDesc(word *wordDesc) *pb.WordDesc {
// 	if word == nil {
// 		return &pb.WordDesc{Err: "word not found"}
// 	}
// 	definitions := make([]*pb.Definition, 0, len(word.Definitions))
// 	for _, def := range word.Definitions {
// 		definitions = append(definitions, &pb.Definition{
// 			Pos:     def.Pos,
// 			Meaning: append([]string{}, def.Meanings...),
// 		})
// 	}
// 	phrases := make([]*pb.Phrase, 0, len(word.Phrases))
// 	for _, phrase := range word.Phrases {
// 		phrases = append(phrases, &pb.Phrase{
// 			Example:   phrase.Example,
// 			ExampleCn: phrase.Example_cn,
// 		})
// 	}
// 	return &pb.WordDesc{
// 		Err:           word.Err,
// 		Word:          word.Word,
// 		Pronunciation: word.Pronunciation,
// 		Definitions:   definitions,
// 		Derivatives:   append([]string{}, word.Derivatives...),
// 		ExamTags:      append([]string{}, word.Exam_tags...),
// 		Example:       word.Example,
// 		ExampleCn:     word.Example_cn,
// 		Phrases:       phrases,
// 		Synonyms:      append([]string{}, word.Synonyms...),
// 		Source:        int32(word.Source),
// 		WordId:        word.WordID,
// 	}
// }
