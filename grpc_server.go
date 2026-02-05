package main

import (
	"context"
	"net"
	"strings"

	pb "github.com/byBit-ovo/coral_word/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type coralWordGrpcServer struct {
	pb.UnimplementedCoralWordServiceServer
}

func RunGrpcServer(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	stopRegister, err := RegisterGrpcToEtcd(addr)
	if err != nil {
		_ = listener.Close()
		return err
	}
	defer func() {
		_ = stopRegister()
	}()
	server := grpc.NewServer()
	pb.RegisterCoralWordServiceServer(server, &coralWordGrpcServer{})
	return server.Serve(listener)
}

func (s *coralWordGrpcServer) QueryWord(ctx context.Context, req *pb.WordRequest) (*pb.WordDescList, error) {
	if req == nil || strings.TrimSpace(req.Word) == "" {
		return nil, status.Error(codes.InvalidArgument, "word is empty")
	}
	wordDescs, err, missWords := QueryWords(req.Word)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query word error: %v", err)
	}
	return &pb.WordDescList{
		WordDescs: toPbWordDescs(wordDescs),
		MissWords: missWords,
		Err:       "false",
		Message: "success",
	}, nil
}

func toPbWordDescs(wordDescs map[string]*wordDesc) map[string]*pb.WordDesc {
	pbWordDescs := make(map[string]*pb.WordDesc, len(wordDescs))
	for word, desc := range wordDescs {
		pbWordDescs[word] = toPbWordDesc(desc)
	}
	return pbWordDescs
}

func toPbWordDesc(word *wordDesc) *pb.WordDesc {
	if word == nil {
		return &pb.WordDesc{Err: "word not found"}
	}
	definitions := make([]*pb.Definition, 0, len(word.Definitions))
	for _, def := range word.Definitions {
		definitions = append(definitions, &pb.Definition{
			Pos:     def.Pos,
			Meaning: append([]string{}, def.Meanings...),
		})
	}
	phrases := make([]*pb.Phrase, 0, len(word.Phrases))
	for _, phrase := range word.Phrases {
		phrases = append(phrases, &pb.Phrase{
			Example:   phrase.Example,
			ExampleCn: phrase.Example_cn,
		})
	}
	return &pb.WordDesc{
		Err:           word.Err,
		Word:          word.Word,
		Pronunciation: word.Pronunciation,
		Definitions:   definitions,
		Derivatives:   append([]string{}, word.Derivatives...),
		ExamTags:      append([]string{}, word.Exam_tags...),
		Example:       word.Example,
		ExampleCn:     word.Example_cn,
		Phrases:       phrases,
		Synonyms:      append([]string{}, word.Synonyms...),
		LlmModelName:  word.LLMModelName,
		WordId:        word.WordID,
		SelectedNotes: word.SelectedNotes,
	}
}

// FromPbWordDesc 将 gRPC 返回的 pb.WordDesc 转为内部 wordDesc，便于存储或业务逻辑
func FromPbWordDesc(p *pb.WordDesc) *wordDesc {
	if p == nil {
		return nil
	}
	defs := make([]Definition, 0, len(p.Definitions))
	for _, d := range p.Definitions {
		defs = append(defs, Definition{
			Pos:      d.Pos,
			Meanings: append([]string{}, d.Meaning...),
		})
	}
	phrases := make([]Phrase, 0, len(p.Phrases))
	for _, ph := range p.Phrases {
		phrases = append(phrases, Phrase{
			Example:    ph.Example,
			Example_cn: ph.ExampleCn,
		})
	}
	return &wordDesc{
		Err:           p.Err,
		Word:          p.Word,
		Pronunciation: p.Pronunciation,
		Definitions:   defs,
		Derivatives:   append([]string{}, p.Derivatives...),
		Exam_tags:     append([]string{}, p.ExamTags...),
		Example:       p.Example,
		Example_cn:    p.ExampleCn,
		Phrases:       phrases,
		Synonyms:      append([]string{}, p.Synonyms...),
		LLMModelName:  p.LlmModelName,
		WordID:        p.WordId,
		SelectedNotes: p.SelectedNotes,
	}
}

// FromPbWordDescList 将 gRPC 返回的 WordDescList 转为 map[string]*wordDesc，便于存储
func FromPbWordDescList(list *pb.WordDescList) map[string]*wordDesc {
	if list == nil || list.WordDescs == nil {
		return nil
	}
	out := make(map[string]*wordDesc, len(list.WordDescs))
	for word, p := range list.WordDescs {
		out[word] = FromPbWordDesc(p)
	}
	return out
}
