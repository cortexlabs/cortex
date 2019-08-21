import tensorflow as tf
import tensorflow_hub as hub
from tensorflow.python.saved_model.signature_def_utils_impl import predict_signature_def

export_dir = "/model/15692343342"
builder = tf.saved_model.builder.SavedModelBuilder(export_dir)
with tf.Session(graph=tf.Graph()) as sess:
    module = hub.Module("https://tfhub.dev/google/imagenet/inception_v3/classification/3") 
    input_params = module.get_input_info_dict()
    image_input = tf.placeholder(name='images', dtype=input_params['images'].dtype, 
        shape=input_params['images'].get_shape())
    sess.run([tf.global_variables_initializer(), tf.tables_initializer()])
    classes = module(image_input)
    
    signature = predict_signature_def(inputs={'images': image_input},
        outputs={'classes': classes})

    builder.add_meta_graph_and_variables(sess,
                                       ["serve"],
                                       signature_def_map={"predict": signature},
                                       strip_default_attrs=True)

builder.save()