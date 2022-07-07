import argparse
import os
import onnxruntime as rt

if __name__ == "__main__":
    # Initialize parser
    parser = argparse.ArgumentParser()

    # Adding optional argument
    parser.add_argument("-f", "--file", help="Model File path")

    # Read arguments from command line
    args = parser.parse_args()
    model_file_path = args.file
    if not model_file_path or not os.path.exists(model_file_path):
        exit(0)
    sess = rt.InferenceSession(model_file_path)

    if len(sess.get_inputs()[0].shape) == 4:
        input_shape = sess.get_inputs()[0].shape[2:]
    elif len(sess.get_inputs()[0].shape) == 3:
        input_shape = sess.get_inputs()[0].shape[1:]
    else:
        input_shape = sess.get_inputs()[0].shape

    if len(sess.get_outputs()[0].shape) == 3:
        output_shape = sess.get_outputs()[0].shape[2:]
    elif len(sess.get_outputs()[0].shape) == 2:
        output_shape = sess.get_outputs()[0].shape[1:]
    else:
        output_shape = sess.get_outputs()[0].shape

    if not str(output_shape[0]).isnumeric():
        output_shape = [-1]

    print(",".join([sess.get_inputs()[0].name, str(input_shape[0]), str(input_shape[1]), sess.get_outputs()[0].name,
                    str(output_shape[0])]))
