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
    console.log(this.data);
  }
}
