/*********************************************************************************
*     File Name           :     ann.go
*     Created By          :     YIMMON, yimmon.zhuang@gmail.com
*     Creation Date       :     [2014-04-22 17:16]
*     Last Modified       :     [2014-06-16 20:40]
*     Description         :
**********************************************************************************/

package ann

/*
#cgo LDFLAGS: -lfann -lm
#include "stdlib.h"
#include "string.h"
#include "fann.h"
typedef void (FANN_API *get_train_data_callback_type)( unsigned int, unsigned int, unsigned int, fann_type * , fann_type * );
extern void get_train_data_callback(unsigned int, unsigned int, unsigned int, fann_type *, fann_type *);

static inline void copy_fann_type_array(const fann_type *src, fann_type *dest, unsigned int n){
    memcpy(dest, src, sizeof(fann_type)*n);
}

static inline int get_fann_type_size(){
    return sizeof(fann_type);
}

static inline int get_uint_size(){
    return sizeof(unsigned int);
}
*/
import "C"

import (
    "math/rand"
    "reflect"
    "time"
    "unsafe"
)

type Type float32
type Ann struct {
    object *C.struct_fann
    output []Type
}
type TrainData struct {
    object *C.struct_fann_train_data
}
type GetTrainDataCallback func(num, numInput, numOutput uint, input, output []Type)
type TrainingAlgorithm C.enum_fann_train_enum
type ActivationFunc C.enum_fann_activationfunc_enum

const (
    TRAIN_INCREMENTAL TrainingAlgorithm = C.FANN_TRAIN_INCREMENTAL
    TRAIN_BATCH       TrainingAlgorithm = C.FANN_TRAIN_BATCH
    TRAIN_RPROP       TrainingAlgorithm = C.FANN_TRAIN_RPROP
    TRAIN_QUICKPROP   TrainingAlgorithm = C.FANN_TRAIN_QUICKPROP

    LINEAR                     ActivationFunc = C.FANN_LINEAR
    THRESHOLD                  ActivationFunc = C.FANN_THRESHOLD
    THRESHOLD_SYMMETRIC        ActivationFunc = C.FANN_THRESHOLD_SYMMETRIC
    SIGMOID                    ActivationFunc = C.FANN_SIGMOID
    SIGMOID_STEPWISE           ActivationFunc = C.FANN_SIGMOID_STEPWISE
    SIGMOID_SYMMETRIC          ActivationFunc = C.FANN_SIGMOID_SYMMETRIC
    SIGMOID_SYMMETRIC_STEPWISE ActivationFunc = C.FANN_SIGMOID_SYMMETRIC_STEPWISE
    GAUSSIAN                   ActivationFunc = C.FANN_GAUSSIAN
    GAUSSIAN_SYMMETRIC         ActivationFunc = C.FANN_GAUSSIAN_SYMMETRIC
    GAUSSIAN_STEPWISE          ActivationFunc = C.FANN_GAUSSIAN_STEPWISE
    ELLIOT                     ActivationFunc = C.FANN_ELLIOT
    ELLIOT_SYMMETRIC           ActivationFunc = C.FANN_ELLIOT_SYMMETRIC
    LINEAR_PIECE               ActivationFunc = C.FANN_LINEAR_PIECE
    LINEAR_PIECE_SYMMETRIC     ActivationFunc = C.FANN_LINEAR_PIECE_SYMMETRIC
    SIN_SYMMETRIC              ActivationFunc = C.FANN_SIN_SYMMETRIC
    COS_SYMMETRIC              ActivationFunc = C.FANN_COS_SYMMETRIC
    SIN                        ActivationFunc = C.FANN_SIN
    COS                        ActivationFunc = C.FANN_COS
)

var (
    getTrainDataCallbackChan = make(chan GetTrainDataCallback, 1)
)

func typeSlice(ptr *C.fann_type, length int) []Type {
    shr := reflect.SliceHeader{
        Data: uintptr(unsafe.Pointer(ptr)),
        Len:  length,
        Cap:  length,
    }
    return *(*[]Type)(unsafe.Pointer(&shr))
}

func CreateStandard(numLayer uint, neuron ...uint) (ann *Ann) {
    ann = new(Ann)
    ann.object = C.fann_create_standard_array(C.uint(numLayer), (*C.uint)(unsafe.Pointer(&neuron[0])))
    return
}

func CreateSparse(connectionRate float32, numLayer uint, neuron ...uint) (ann *Ann) {
    ann = new(Ann)
    ann.object = C.fann_create_sparse_array(C.float(connectionRate), C.uint(numLayer), (*C.uint)(unsafe.Pointer(&neuron[0])))
    return
}

func CreateShortcut(numLayer uint, neuron ...uint) (ann *Ann) {
    ann = new(Ann)
    ann.object = C.fann_create_shortcut_array(C.uint(numLayer), (*C.uint)(unsafe.Pointer(&neuron[0])))
    return
}

func CreateFromFile(filepath string) (ann *Ann) {
    cstr := C.CString(filepath)
    defer C.free(unsafe.Pointer(cstr))
    ann = new(Ann)
    ann.object = C.fann_create_from_file(cstr)
    return ann
}

//export get_train_data_callback
func get_train_data_callback(num, numInput, numOutput C.uint, input, output *C.fann_type) {
    callback := <-getTrainDataCallbackChan
    callback(uint(num), uint(numInput), uint(numOutput), typeSlice(input, int(numInput)), typeSlice(output, int(numOutput)))
    getTrainDataCallbackChan <- callback
}

func CreateTrainFromCallback(numData, numInput, numOutput uint, callback GetTrainDataCallback) *TrainData {
    trainData := new(TrainData)
    getTrainDataCallbackChan <- callback
    trainData.object = C.fann_create_train_from_callback(C.uint(numData), C.uint(numInput), C.uint(numOutput),
        (C.get_train_data_callback_type)(C.get_train_data_callback))
    <-getTrainDataCallbackChan
    return trainData
}

func (self *Ann) Destroy() {
    C.fann_destroy(self.object)
}

func (self *Ann) Run(input []Type) []Type {
    coutput := C.fann_run(self.object, (*C.fann_type)(&input[0]))
    n := int(C.fann_get_num_output(self.object))
    if len(self.output) < n {
        self.output = make([]Type, n)
    } else {
        self.output = self.output[:n]
    }
    C.copy_fann_type_array(coutput, (*C.fann_type)(&self.output[0]), C.uint(uint(n)))
    return self.output
}

func (self *Ann) RandomizeWeights(min, max Type) {
    C.fann_randomize_weights(self.object, C.fann_type(min), C.fann_type(max))
}

func (self *Ann) PrintParameters() {
    C.fann_print_parameters(self.object)
}

func (self *Ann) GetNumInput() uint {
    return uint(C.fann_get_num_input(self.object))
}

func (self *Ann) GetNumOutput() uint {
    return uint(C.fann_get_num_output(self.object))
}

func (self *Ann) GetMSE() float32 {
    return float32(C.fann_get_MSE(self.object))
}

func (self *Ann) GetBitFail() uint {
    return uint(C.fann_get_bit_fail(self.object))
}

func (self *Ann) ResetMSE() {
    C.fann_reset_MSE(self.object)
}

func (self *Ann) TrainOnData(data *TrainData, maxEpochs, epochsBetweenReports uint, desiredError float32) {
    C.fann_train_on_data(self.object, data.object, C.uint(maxEpochs), C.uint(epochsBetweenReports), C.float(desiredError))
}

func (self *Ann) CascadetrainOnData(data *TrainData, maxEpochs, epochsBetweenReports uint, desiredError float32) {
    C.fann_cascadetrain_on_data(self.object, data.object, C.uint(maxEpochs), C.uint(epochsBetweenReports), C.float(desiredError))
}

func (self *Ann) TestData(data *TrainData) float32 {
    return float32(C.fann_test_data(self.object, data.object))
}

func (self *Ann) ScaleTrain(data *TrainData) {
    C.fann_scale_train(self.object, data.object)
}

func (self *Ann) SetBitFailLimit(bitFailLimit Type) {
    C.fann_set_bit_fail_limit(self.object, C.fann_type(bitFailLimit))
}

func (self *Ann) SetTrainingAlgorithm(trainingAlgorithm TrainingAlgorithm) {
    C.fann_set_training_algorithm(self.object, C.enum_fann_train_enum(trainingAlgorithm))
}

func (self *Ann) SetActivationFunctionLayer(activationFunc ActivationFunc, layer int) {
    C.fann_set_activation_function_layer(self.object, C.enum_fann_activationfunc_enum(activationFunc), C.int(layer))
}

func (self *Ann) SetActivationFunctionHidden(activationFunc ActivationFunc) {
    C.fann_set_activation_function_hidden(self.object, C.enum_fann_activationfunc_enum(activationFunc))
}

func (self *Ann) SetActivationFunctionOutput(activationFunc ActivationFunc) {
    C.fann_set_activation_function_output(self.object, C.enum_fann_activationfunc_enum(activationFunc))
}

func (self *Ann) SetActivationSteepnessLayer(steepness Type, layer int) {
    C.fann_set_activation_steepness_layer(self.object, C.fann_type(steepness), C.int(layer))
}

func (self *Ann) SetActivationSteepnessHidden(steepness Type) {
    C.fann_set_activation_steepness_hidden(self.object, C.fann_type(steepness))
}

func (self *Ann) SetActivationSteepnessOutput(steepness Type) {
    C.fann_set_activation_steepness_output(self.object, C.fann_type(steepness))
}

func (self *Ann) Save(filepath string) int {
    cstr := C.CString(filepath)
    defer C.free(unsafe.Pointer(cstr))
    return int(C.fann_save(self.object, cstr))
}

func (self *TrainData) Destroy() {
    C.fann_destroy_train(self.object)
}

func (self *TrainData) Shuffle() {
    C.fann_shuffle_train_data(self.object)
}

func init() {
    /*
       if int(unsafe.Sizeof(Type(0))) != int(C.get_fann_type_size()) {
           panic("Go Type NOT equal to C.fann_type")
       }
       if int(unsafe.Sizeof(uint(0))) != int(C.get_uint_size()) {
           panic("Go uint NOT equal to C.uint")
       }
    */
    rand.Seed(time.Now().Unix())
}
