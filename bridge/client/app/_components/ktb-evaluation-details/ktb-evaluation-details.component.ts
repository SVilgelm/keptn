import * as Highcharts from "highcharts";

import * as moment from 'moment';
import {ChangeDetectorRef, Component, Input, OnInit} from '@angular/core';
import {DtChartSeriesVisibilityChangeEvent} from "@dynatrace/barista-components/chart";

import {DataService} from "../../_services/data.service";
import DateUtil from "../../_utils/date.utils";

@Component({
  selector: 'ktb-evaluation-details',
  templateUrl: './ktb-evaluation-details.component.html',
  styleUrls: ['./ktb-evaluation-details.component.scss']
})
export class KtbEvaluationDetailsComponent implements OnInit {

  public _evaluationData: any;
  public _evaluationSource: string;

  public _selectedEvaluationData: any;

  public _view: string = "singleevaluation";

  public _chartOptions: Highcharts.Options = {
    xAxis: {
      type: 'datetime',
    },
    yAxis: [
      {
        title: null,
        labels: {
          format: '{value}',
        },
        min: 0,
        max: 100,
        tickInterval: 10,
      },
      {
        title: null,
        labels: {
          format: '{value}',
        },
        opposite: true,
        tickInterval: 50,
      },
    ],
    plotOptions: {
      column: {
        stacking: 'normal',
        pointWidth: 5,
        minPointLength: 2,
        point: {
          events: {
            click: (event) => {
              this._chartSeriesClicked(event);
              return true;
            }
          }
        },
      },
    },
  };

  public _chartSeries: Highcharts.IndividualSeriesOptions[] = [
    {
      name: 'Evaluation passed',
      type: 'column',
      data: [],
      color: '#006bb8',
      cursor: 'pointer'
    },
    {
      name: 'Evaluation failed',
      type: 'column',
      data: [],
      color: '#c41425',
      cursor: 'pointer'
    },
  ];

  @Input()
  get evaluationData(): any {
    return this._evaluationData;
  }
  set evaluationData(evaluationData: any) {
    if (this._evaluationData !== evaluationData) {
      this._evaluationData = evaluationData;
      this._changeDetectorRef.markForCheck();
    }
  }

  @Input()
  get evaluationSource(): any {
    return this._evaluationSource;
  }
  set evaluationSource(evaluationSource: any) {
    if (this._evaluationSource !== evaluationSource) {
      this._evaluationSource = evaluationSource;
      this._changeDetectorRef.markForCheck();
    }
  }

  constructor(private _changeDetectorRef: ChangeDetectorRef, private dataService: DataService) { }

  ngOnInit() {
    this.dataService.evaluationResults.subscribe((evaluationData) => {
      if(this.evaluationData === evaluationData) {
        this.updateChartData(evaluationData.evaluationHistory);
        this._changeDetectorRef.markForCheck();
      }
    });
  }

  updateChartData(evaluationHistory) {
    let chartSeries = [];

    let evaluationPassed = [];
    let evaluationFailed = [];

    evaluationHistory.forEach((evaluation) => {
      let data = {
        x: moment(evaluation.time).unix()*1000,
        y: evaluation.data.evaluationdetails ? evaluation.data.evaluationdetails.score : 0,
        evaluationData: evaluation
      };
      if(evaluation.data.result == 'pass')
        evaluationPassed.push(data);
      else
        evaluationFailed.push(data);

      evaluation.data.evaluationdetails.indicatorResults.forEach((indicatorResult) => {
        let indicatorData = {
          x: moment(evaluation.time).unix()*1000,
          y: indicatorResult.value.value,
          indicatorResult: indicatorResult
        };
        let indicatorChartSeries = chartSeries.find(series => series.name == indicatorResult.value.metric);
        if(!indicatorChartSeries) {
          indicatorChartSeries = {
            name: indicatorResult.value.metric,
            type: 'line',
            yAxis: 1,
            data: [],
          };
          chartSeries.push(indicatorChartSeries);
        }
        indicatorChartSeries.data.push(indicatorData);
      });
    });
    this._chartSeries = [
      {
        name: 'Evaluation passed',
        type: 'column',
        data: evaluationPassed,
        color: '#7dc540',
        cursor: 'pointer'
      },
      {
        name: 'Evaluation failed',
        type: 'column',
        data: evaluationFailed,
        color: '#c41425',
        cursor: 'pointer'
      },
      ...chartSeries
    ];
  }

  switchEvaluationView(event) {
    this._view = this._view == "singleevaluation" ? "evaluationcomparison" : "singleevaluation";
    if(this._view == "evaluationcomparison") {
      this.dataService.loadEvaluationResults(this._evaluationData, this._evaluationSource);
      this._changeDetectorRef.markForCheck();
    }
  }

  seriesVisibilityChanged(_: DtChartSeriesVisibilityChangeEvent): void {
    // NOOP
  }

  _chartSeriesClicked(event): boolean {
    this._selectedEvaluationData = event.point.evaluationData.data;
    return true;
  }

  getCalendarFormat() {
    return DateUtil.getCalendarFormats().sameElse;
  }

  log(tooltip){
    console.log("tooltip", tooltip);
    return tooltip.points;
  }

}
