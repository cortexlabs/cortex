from concurrent import futures
import grpc

import iris_classifier_pb2
import iris_classifier_pb2_grpc

class PredictorServicer(iris_classifier_pb2_grpc.PredictorServicer):
    def __init__(self, predictor_impl):
        self.predictor_impl = predictor_impl

    def Predict(self, payload, context):
        return iris_classifier_pb2.Response(classification="setosa")


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=5))
    iris_classifier_pb2_grpc.add_PredictorServicer_to_server(
        PredictorServicer(None), server)
    server.add_insecure_port('localhost:50000')
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
