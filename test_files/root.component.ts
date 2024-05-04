import {ChangeDetectionStrategy, Component} from '@angular/core';

@Component({
  changeDetection: ChangeDetectionStrategy.OnPush,
  selector: 'tg-root',
  templateUrl: './root.component.pug'
})

export class RootComponent {
  public data: number = 23;

  public get getter(): number {
    return this.data;
  }

  constructor() {
    RootComponent.prototype.prototypalUsage
    console.log(this.connor)
    console.log(this.data);
    console.log(this.not);
  }
}
