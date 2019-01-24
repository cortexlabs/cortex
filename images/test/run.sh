#!/bin/bash
err=0
trap 'err=1' ERR

cd spark_job
pytest
cd ..

cd lib
pytest
cd ..

test $err = 0
