import numpy as np
from keras.layers import Input, Conv2D, BatchNormalization, MaxPooling2D, Reshape, Dense, LSTM, add, concatenate, Dropout, Lambda, Flatten
from keras.models import Model
import keras.backend as K
from stn.spatial_transformer import SpatialTransformer

def loc_net(input_shape):
    b = np.zeros((2, 3), dtype='float32')
    b[0, 0] = 1
    b[1, 1] = 1
    w = np.zeros((64, 6), dtype='float32')
    weights = [w, b.flatten()]

    loc_input = Input(input_shape)

    loc_conv_1 = Conv2D(16, (5, 5), padding='same', activation='relu')(loc_input)
    loc_conv_2 = Conv2D(32, (5, 5), padding='same', activation='relu')(loc_conv_1)
    loc_fla = Flatten()(loc_conv_2)
    loc_fc_1 = Dense(64, activation='relu')(loc_fla)
    loc_fc_2 = Dense(6, weights=weights)(loc_fc_1)

    output = Model(inputs=loc_input, outputs=loc_fc_2)

    return output


def ctc_lambda_func(args):
    iy_pred, ilabels, iinput_length, ilabel_length = args
    # the 2 is critical here since the first couple outputs of the RNN
    # tend to be garbage:
    iy_pred = iy_pred[:, 2:, :]  # no such influence
    return K.ctc_batch_cost(ilabels, iy_pred, iinput_length, ilabel_length)


def CRNN_STN(cfg):

    inputs = Input((cfg.width, cfg.height, cfg.nb_channels))
    c_1 = Conv2D(cfg.conv_filter_size[0], (3, 3), activation='relu', padding='same', name='conv_1')(inputs)
    c_2 = Conv2D(cfg.conv_filter_size[1], (3, 3), activation='relu', padding='same', name='conv_2')(c_1)
    c_3 = Conv2D(cfg.conv_filter_size[2], (3, 3), activation='relu', padding='same', name='conv_3')(c_2)
    bn_3 = BatchNormalization(name='bn_3')(c_3)
    p_3 = MaxPooling2D(pool_size=(2, 2), name='maxpool_3')(bn_3)

    c_4 = Conv2D(cfg.conv_filter_size[3], (3, 3), activation='relu', padding='same', name='conv_4')(p_3)
    c_5 = Conv2D(cfg.conv_filter_size[4], (3, 3), activation='relu', padding='same', name='conv_5')(c_4)
    bn_5 = BatchNormalization(name='bn_5')(c_5)
    p_5 = MaxPooling2D(pool_size=(2, 2), name='maxpool_5')(bn_5)

    c_6 = Conv2D(cfg.conv_filter_size[5], (3, 3), activation='relu', padding='same', name='conv_6')(p_5)
    c_7 = Conv2D(cfg.conv_filter_size[6], (3, 3), activation='relu', padding='same', name='conv_7')(c_6)
    bn_7 = BatchNormalization(name='bn_7')(c_7)

    bn_7_shape = bn_7.get_shape()
    loc_input_shape = (bn_7_shape[1].value, bn_7_shape[2].value, bn_7_shape[3].value)
    stn = SpatialTransformer(localization_net=loc_net(loc_input_shape), output_size=(loc_input_shape[0], loc_input_shape[1]))(bn_7)

    reshape = Reshape(target_shape=(int(bn_7_shape[1]), int(bn_7_shape[2] * bn_7_shape[3])), name='reshape')(stn)

    fc_9 = Dense(cfg.lstm_nb_units[0], activation='relu', name='fc_9')(reshape)

    lstm_10 = LSTM(cfg.lstm_nb_units[0], kernel_initializer="he_normal", return_sequences=True, name='lstm_10')(fc_9)
    lstm_10_back = LSTM(cfg.lstm_nb_units[0], kernel_initializer="he_normal", go_backwards=True, return_sequences=True, name='lstm_10_back')(fc_9)
    lstm_10_add = add([lstm_10, lstm_10_back])

    lstm_11 = LSTM(cfg.lstm_nb_units[1], kernel_initializer="he_normal", return_sequences=True, name='lstm_11')(lstm_10_add)
    lstm_11_back = LSTM(cfg.lstm_nb_units[1], kernel_initializer="he_normal", go_backwards=True, return_sequences=True, name='lstm_11_back')(lstm_10_add)
    lstm_11_concat = concatenate([lstm_11, lstm_11_back])
    do_11 = Dropout(cfg.dropout_rate, name='dropout')(lstm_11_concat)

    fc_12 = Dense(len(cfg.characters), kernel_initializer='he_normal', activation='softmax', name='fc_12')(do_11)

    prediction_model = Model(inputs=inputs, outputs=fc_12)

    labels = Input(name='labels', shape=[cfg.label_len], dtype='float32')
    input_length = Input(name='input_length', shape=[1], dtype='int64')
    label_length = Input(name='label_length', shape=[1], dtype='int64')

    ctc_loss = Lambda(ctc_lambda_func, output_shape=(1,), name='ctc')([fc_12, labels, input_length, label_length])
    
    training_model = Model(inputs=[inputs, labels, input_length, label_length], outputs=[ctc_loss]) 

    return training_model, prediction_model