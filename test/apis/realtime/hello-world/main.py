import os
import time
import threading as td

from flask import Flask

app = Flask(__name__)


@app.route("/")
def hello_world():
    time.sleep(1)
    msg = f"Hello World! (TID={td.get_ident()}"
    print(msg)
    return msg


if __name__ == "__main__":
    app.run(debug=True, host="0.0.0.0", port=int(os.getenv("CORTEX_PORT", "8080")))
