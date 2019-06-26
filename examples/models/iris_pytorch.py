"""
Requirements.txt

scikit-learn
torch
"""
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, precision_score, recall_score
from sklearn.datasets import load_iris
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.autograd import Variable

iris = load_iris()
X, y = iris.data, iris.target


class Net(nn.Module):
    # define nn
    def __init__(self):
        super(Net, self).__init__()
        self.fc1 = nn.Linear(4, 100)
        self.fc2 = nn.Linear(100, 100)
        self.fc3 = nn.Linear(100, 3)
        self.softmax = nn.Softmax(dim=1)

    def forward(self, X):
        X = F.relu(self.fc1(X))
        X = self.fc2(X)
        X = self.fc3(X)
        X = self.softmax(X)

        return X


X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

# wrap up with Variable in pytorch
train_X = Variable(torch.Tensor(X_train).float())
test_X = Variable(torch.Tensor(X_test).float())
train_y = Variable(torch.Tensor(y_train).long())
test_y = Variable(torch.Tensor(y_test).long())

net = Net()

criterion = nn.CrossEntropyLoss()  # cross entropy loss

optimizer = torch.optim.SGD(net.parameters(), lr=0.01)

for epoch in range(1000):
    optimizer.zero_grad()
    out = net(train_X)
    loss = criterion(out, train_y)
    loss.backward()
    optimizer.step()

    if epoch % 100 == 0:
        print("number of epoch {} loss {}".format(epoch, loss))

predict_out = net(test_X)
_, predict_y = torch.max(predict_out, 1)

print("prediction accuracy {}".format(accuracy_score(test_y.data, predict_y.data)))

dummy_input = torch.randn(1, 4)

torch.onnx.export(
    net,
    dummy_input,
    "iris_pytorch.onnx",
    verbose=True,
    input_names=["input"],
    output_names=["species"],
)
