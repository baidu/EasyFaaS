/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/brn"
	"github.com/baidu/openless/pkg/controller/rtctrl"
	innerErr "github.com/baidu/openless/pkg/error"
	"github.com/baidu/openless/pkg/util/json"
)

type ControllerInterface interface {
	NewClients(string) *Clients
	Do(ctx *InvokeContext)
}

// TODO: error
var (
	NoOutputMsg = "no output"
)

func (controller *Controller) Do(ctx *InvokeContext) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			var buf [4096]byte
			runtime.Stack(buf[:], true)
			ctx.Logger.Errorf("Panic recovered in do, %+v | %s", r, string(buf[:]))
			err = fmt.Errorf("%+v", r)
		}
		if err != nil {
			buildErrorResponse(ctx, err)
		}
	}()
	defer controller.SummaryMetrics(ctx)

	ctx.Logger.V(3).Infof("start to invoke request %s", ctx.RequestID)

	if _, err = controller.getFunction(ctx); err != nil {
		return
	}

	if _, err = controller.getRuntimeConfiguration(ctx); err != nil {
		return
	}

	if err = controller.getRuntime(ctx); err != nil {
		return
	}
	defer controller.putRuntime(ctx)

	controller.invocation(ctx)
	controller.buildResponse(ctx, ctx.Output.Output)
}

func buildErrorResponse(ctx *InvokeContext, err error) {
	var status int
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		defer func() {
			ctx.Logger.V(9).Infof("user function response code %d", status)
			ctx.Metrics.SetLabel(responseCodeLabel, strconv.Itoa(status))
		}()
	}

	finalErr := innerErr.GenericKunFinalError(err)
	ctx.Response.SetStatusCode(finalErr.Status)
	status = finalErr.Status
	ctx.Response.SetHeader(api.XBceFunctionError, "Unhandled")
	errMsg := finalErr.Error()
	ctx.Response.SetBody([]byte(errMsg))
}

func parseBrn(ctx *InvokeContext) (brn.BRN, error) {
	if functionBRN, err := brn.Parse(ctx.FunctionBRN); err != nil {
		return functionBRN, err
	} else {
		ctx.CallerUser = &api.User{
			ID: functionBRN.AccountID,
		}
		resources := strings.Split(functionBRN.Resource, ":")
		// eg: function:hello-tmp:$LATEST
		if len(resources) == 3 {
			ctx.FunctionName = resources[1]
			ctx.Qualifier = resources[2]
		}
		return functionBRN, nil
	}
}

func (controller *Controller) getFunction(ctx *InvokeContext) (hitCache bool, err error) {
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics.StepStart(StageGetFunction)
		ctx.Metrics.StepStart(StageGetFunctionHitCache)
		defer func() {
			if hitCache {
				ctx.Metrics.StepDone(StageGetFunctionHitCache)
				ctx.Metrics.ScrapStage(StageGetFunction)
			} else {
				ctx.Metrics.StepDone(StageGetFunction)
				ctx.Metrics.ScrapStage(StageGetFunctionHitCache)
			}
			ctx.Metrics.SetLabel(getFunctionBrnLabel, ctx.FunctionBRN)
			ctx.Metrics.SetLabel(getFunctionHitCacheLabel, strconv.FormatBool(hitCache))
		}()
	}

	ctx.Logger.V(9).Info("get function configuration")
	hitCache = false

	input := api.GetFunctionInput{
		Authorization: ctx.Authorization,
		RequestID:     ctx.RequestID,
		AccountID:     ctx.AccountID,
		WithCache:     true,
	}

	if ctx.InvokeType == api.InvokeTypeEvent || ctx.InvokeType == api.InvokeTypeHttpTrigger || ctx.InvokeType == api.InvokeTypeMqhub {
		input.SimpleAuth = true
	}
	if ctx.TriggerType == api.TriggerTypeVscode {
		input.SimpleAuth = true
	}

	if ctx.FunctionBRN != "" {
		hitCache, err = controller.getFunctionByFunctionBrn(ctx, &input)
	} else {
		hitCache, err = controller.getFunctionByFunctionName(ctx, &input)
	}
	if err != nil {
		return false, err
	}

	ctx.OwnerUser = &api.User{
		ID: ctx.Function.Configuration.Uid,
	}

	// TODO: implement the authenticate policies

	// TODO: or caller user has access to invoke function？
	if controller.runOptions.SimpleAuth {
		if ctx.AccountID != ctx.OwnerUser.ID {
			ctx.Logger.Warnf("invalid caller id, owner id %s caller id %s", ctx.OwnerUser.ID, ctx.AccountID)
			err = innerErr.NewInvalidInvokeCallerException(fmt.Sprintf("owner id %s caller id %s", ctx.OwnerUser.ID, ctx.AccountID), nil)
			return
		}
	}

	ctx.Logger.Debugf("function_output=%v err=%v", ctx.Function, err)
	ctx.Logger.V(8).Infof("function_brn=%s", *ctx.Function.Configuration.FunctionArn)

	return hitCache, nil
}

func (controller *Controller) getFunctionByFunctionName(ctx *InvokeContext, input *api.GetFunctionInput) (hitCache bool, err error) {
	hitCache = false
	// do not support for alias qualifier
	if !api.RegVersion.MatchString(ctx.Qualifier) {
		err = innerErr.NewInvalidParameterValueException("invalid function qualifier", nil)
		return false, err
	}

	// Do not use cache, when invoked by function name
	input.WithCache = false

	input.SetFunctionName(ctx.FunctionName).SetQualifier(ctx.Qualifier)
	ctx.Logger.Debugf("function_input=%v", input)

	defer ctx.Logger.TimeTrack(time.Now(), "GetFunction",
		zap.String("function_name", *input.FunctionName),
		zap.String("qualifier", *input.Qualifier),
	)

	if ctx.Function, hitCache, err = ctx.Clients.DataStorer.GetFunction(input); err != nil {
		ctx.Logger.Debugf("function store get failed, err(%+v)", err)
		return hitCache, err
	}

	return hitCache, nil
}

func (controller *Controller) getFunctionByFunctionBrn(ctx *InvokeContext, input *api.GetFunctionInput) (hitCache bool, err error) {
	hitCache = false
	if ctx.Brn, err = parseBrn(ctx); err != nil {
		return
	}
	// get latest version function without cache
	ctx.Logger.V(6).Infof("brn version %+v", ctx.Qualifier)

	if !api.RegVersion.MatchString(ctx.Qualifier) {
		if !controller.runOptions.EnableCanary {
			err = innerErr.NewInvalidParameterValueException("invalid brn", nil)
			return false, err
		}
		if err := controller.getCanaryFunctionBrn(ctx); err != nil {
			return false, err
		}
	}

	if ctx.Qualifier == "$LATEST" {
		input.WithCache = false
	}
	brnStr := ctx.Brn.String()
	input.SetFunctionName(brnStr).SetQualifier(ctx.Qualifier)
	ctx.Logger.Debugf("function_input=%v", input)

	defer ctx.Logger.TimeTrack(time.Now(), "GetFunction",
		zap.String("function_name", *input.FunctionName),
		zap.String("qualifier", *input.Qualifier),
	)

	if ctx.Function, hitCache, err = ctx.Clients.DataStorer.GetFunction(input); err != nil {
		ctx.Logger.Debugf("function store get failed, err(%+v)", err)
		return hitCache, err
	}

	return hitCache, nil
}

func (controller *Controller) getCanaryFunctionBrn(ctx *InvokeContext) (err error) {
	input := &api.GetAliasInput{
		FunctionBrn:   ctx.FunctionBRN,
		Authorization: ctx.Authorization,
		RequestID:     ctx.RequestID,
		AccountID:     ctx.AccountID,
		WithCache:     true,
		SimpleAuth:    true,
	}
	// TODO: optimize getAlias api ( inside-v1 and openApi)
	// uri & query parameter & response structure is totally different !!!
	// force to use inside api
	alias, _, err := controller.insideDataStorer.GetAlias(input)
	if err != nil {
		ctx.Logger.Debugf("function store get failed, err(%+v)", err)
		return err
	}
	additionalVersion := *alias.AdditionalVersion
	weight := *alias.AdditionalVersionWeight

	if additionalVersion != "" && weight != 0 {
		r := rand.Float64()
		if r > weight {
			ctx.Qualifier = alias.FunctionVersion
		} else {
			ctx.Qualifier = additionalVersion
		}
		ctx.Brn.Resource = "function:" + alias.FunctionName + ":" + ctx.Qualifier
		ctx.FunctionBRN = ctx.Brn.String()
	}

	return nil
}

func (controller *Controller) getRuntimeConfiguration(ctx *InvokeContext) (hitCache bool, err error) {
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics.StepStart(StageGetRuntimeConfiguration)
		ctx.Metrics.StepStart(StageGetRuntimeConfigurationHitCache)
		defer func() {
			if hitCache {
				ctx.Metrics.StepDone(StageGetRuntimeConfigurationHitCache)
				ctx.Metrics.ScrapStage(StageGetRuntimeConfiguration)
			} else {
				ctx.Metrics.StepDone(StageGetRuntimeConfiguration)
				ctx.Metrics.ScrapStage(StageGetRuntimeConfigurationHitCache)
			}
			ctx.Metrics.SetLabel(getConfigurationHitCacheLabel, strconv.FormatBool(hitCache))
		}()
	}

	ctx.Logger.V(9).Info("get runtime configuration")
	hitCache = false

	namep := ctx.Function.Configuration.Runtime
	if namep == nil {
		return hitCache, innerErr.NewServiceException("runtime name is nil pointer", nil)
	}
	name := *namep

	input := &api.GetRuntimeConfigurationInput{
		RuntimeName:   name,
		Authorization: ctx.Authorization,
		RequestID:     ctx.RequestID,
	}
	ctx.Runtime, hitCache, err = ctx.Clients.DataStorer.GetRuntimeConfiguration(input)
	return hitCache, err
}

func (controller *Controller) getRuntime(ctx *InvokeContext) (err error) {
	runtimeT := api.RuntimeViaUnknown
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics.StepStart(StageGetPodWarm)
		ctx.Metrics.StepStart(StageGetPodCold)
		defer func() {
			if runtimeT == api.RuntimeViaWarm {
				ctx.Metrics.StepDone(StageGetPodWarm)
				ctx.Metrics.ScrapStage(StageGetPodCold)
			} else if runtimeT == api.RuntimeViaCold {
				ctx.Metrics.StepDone(StageGetPodCold)
				ctx.Metrics.ScrapStage(StageGetPodWarm)
			}

			ctx.Metrics.SetLabel(podSourceLabel, runtimeT)
		}()
	}

	ctx.Input = &rtctrl.InvocationInput{
		ExternalRequestID: ctx.ExternalRequestID,
		RequestID:         ctx.RequestID,
		User:              ctx.OwnerUser,
		Code:              ctx.Function.Code,
		Configuration:     ctx.Function.Configuration,
		LogConfig:         ctx.Function.LogConfig,
		EnableMetrics:     ctx.RunOptions.RecommendedOptions.Features.EnableMetrics,
		IsLogTail:         ctx.LogType.IsLogTypeTail(),
		Request:           ctx.Request,
		Response:          ctx.Response,
		Logger:            ctx.Logger,
		InvokeType:        ctx.InvokeType,
		TriggerType:       ctx.TriggerType,
	}

	if ctx.WithStreamMode || strings.HasSuffix(ctx.Runtime.Name, "stream") {
		ctx.WithStreamMode = true
		ctx.Input.WithStreamMode = true
	}
	for i := 0; i < 2; i++ {
		runtimeT, err = controller.tryGetRuntime(ctx)
		if err != nil {
			ctx.Logger.Errorf("#%d try to get runtime failed: %s", i, err.Error())
		} else {
			break
		}
	}

	return
}

func (controller *Controller) tryGetRuntime(ctx *InvokeContext) (runtimeType string, err error) {
	runtimeType = api.RuntimeViaUnknown
	var rt *rtctrl.RuntimeInfo
	rt = controller.runtimeDispatcher.FindWarmRuntime(ctx.Input)
	if rt != nil {
		ctx.Input.Runtime = rt
		runtimeType = api.RuntimeViaWarm
		return
	}
	ctx.Logger.Infof("get runtime failed, try to warm up one")

	rt, recommendation := controller.runtimeDispatcher.OccupyColdRuntime(ctx.Input)
	if rt == nil {
		err = innerErr.NewTooManyRequestsException("empty runtime", nil)
		ctx.Logger.V(9).Infof("found empty runtime, all runtime: %s", controller.runtimeDispatcher)
		return
	}

	ctx.Logger.Infof("warm up container %s", rt.RuntimeID)
	input := &api.FuncletClientWarmUpInput{
		ContainerID:          rt.RuntimeID,
		RequestID:            ctx.RequestID,
		Code:                 ctx.Function.Code,
		Configuration:        ctx.Function.Configuration,
		RuntimeConfiguration: ctx.Runtime,
		WithStreamMode:       ctx.WithStreamMode,
	}
	if recommendation != nil {
		input.NeedScaleUp = true
		input.ScaleUpRecommendation = recommendation
	}
	_, err = ctx.Clients.FuncletClient.WarmUp(input)
	if err != nil {
		rt.Invalidate()
		if err := rt.Release(); err != nil {
			ctx.Logger.Errorf("release runtime %s failed: %s", rt.RuntimeID, err)
		}
		err = fmt.Errorf("warm up runtime %s failed: %s", rt.RuntimeID, err.Error())
		return
	}
	ctx.Input.Runtime = rt
	if ctx.Function.Configuration.PodConcurrentQuota == 0 {
		ctx.Input.Runtime.ConcurrentMode = false
	}
	runtimeType = api.RuntimeViaCold
	return
}

func (controller *Controller) buildResponse(ctx *InvokeContext, output *rtctrl.InvocationResponse) {
	status := http.StatusOK
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		defer func() {
			ctx.Logger.V(9).Infof("user function response code %d", status)
			ctx.Metrics.SetLabel(responseCodeLabel, strconv.Itoa(status))
		}()
	}

	if ctx.Output == nil {
		status = http.StatusBadRequest
		ctx.Response.SetStatusCode(http.StatusBadRequest)
		ctx.Response.SetBody([]byte(NoOutputMsg))
		return
	}

	//ctx.Logger.V(6).Info("invoke output",
	//	zap.String("func_result", "(skipped)"),
	//	zap.Int("func_result_len", len(output.FuncResult)),
	//	zap.String("log_message", "(skipped)"),
	//	zap.Int("log_message_len", len(output.LogMessage)),
	//	zap.String("func_error", output.FuncError),
	//	zap.String("error_info", output.ErrorInfo))

	funcError := output.FuncError
	payload := getFuncResult(output)
	var logMessage []string
	if ctx.LogType.IsLogTypeTail() {
		logMessage = output.LogMessage
	}

	ctx.Response.SetStatusCode(http.StatusOK)

	if ctx.Statistic.Statistic != nil {
		ctx.Response.SetHeader(api.HeaderOpenLessExecTime, strconv.FormatFloat(ctx.Statistic.Statistic.Duration, 'f', 3, 64))
	}

	if ctx.Input.EnableMetrics {
		ctx.Metrics.rtCtrl = ctx.Statistic.Metric
	}

	if ctx.WithStreamMode || ctx.InvokeType == api.InvokeTypeEvent {
		return
	}
	if ctx.LogToBody {
		status = buildTotallyToBody(ctx, funcError, payload, logMessage)
	} else {
		status = build(ctx, funcError, payload, logMessage)
	}
	return
}

func buildTotallyToBody(ctx *InvokeContext, funcError, payload string, logMessage []string) (status int) {
	status = http.StatusOK
	result := logToBodyResult{
		FuncError: funcError,
		LogResult: strings.Join(logMessage, ""),
		Payload:   payload,
	}
	if len(funcError) != 0 {
		status = http.StatusInternalServerError
	}
	bodyData, _ := json.Marshal(result)
	ctx.Response.SetHeader("Content-Type", "application/json; charset=utf-8")
	ctx.Response.SetBody(bodyData)
	return
}

func build(ctx *InvokeContext, funcError, payload string, logMessage []string) (status int) {
	status = http.StatusOK
	if len(funcError) > 0 {
		status = http.StatusInternalServerError
		ctx.Response.SetHeader(api.XBceFunctionError, funcError)
	}

	if logMessage != nil && len(logMessage) > 0 {
		logResult := base64.StdEncoding.EncodeToString([]byte(strings.Join(logMessage, "")))
		ctx.Response.SetHeader(api.HeaderLogResult, logResult)
	}

	ctx.Response.SetBody([]byte(payload))
	return
}

func getFuncResult(output *rtctrl.InvocationResponse) string {
	// FuncError
	if len(output.FuncError) == 0 {
		return output.FuncResult
	}
	if output.ErrorInfo == "Invoke timeout." {
		return output.FuncResult
	}

	return output.ErrorInfo
}

func (controller *Controller) putRuntime(ctx *InvokeContext) {
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics.StepStart(StagePutPod)
		defer ctx.Metrics.StepDone(StagePutPod)
	}
	if err := ctx.Input.Runtime.Release(); err != nil {
		ctx.Logger.Errorf("release runtime %s failed: %s", ctx.Input.Runtime.RuntimeID, err)
	}
}

func (controller *Controller) invocation(ctx *InvokeContext) {
	if ctx.RunOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics.StepStart(StageInvocation)
		defer ctx.Metrics.StepDone(StageInvocation)
	}
	ctx.Output = ctx.Clients.RuntimeControl.InvokeFunction(ctx.Input)
	ctx.Statistic = ctx.Output.Statistic
}

func (controller *Controller) SummaryMetrics(ctx *InvokeContext) {
	if ctx.Metrics != nil {
		ctx.Metrics.Overall()
		ctx.Metrics.WriteSummary(controller.runOptions.RecommendedOptions.Features.SummaryOverheadMs)
	}
}
